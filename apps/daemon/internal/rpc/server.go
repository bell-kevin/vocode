package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type ServerOptions struct {
	Logger *log.Logger
	Stdin  io.Reader
	Stdout io.Writer
	Router *Router
}

type Server struct {
	logger *log.Logger
	stdin  io.Reader
	stdout io.Writer
	router *Router

	encoder *json.Encoder

	writeMu sync.Mutex

	outboundMu sync.Mutex
	nextID     int64
	pending    map[int64]chan inboundResponse
	pendingMu  sync.Mutex
}

func NewServer(opts ServerOptions) *Server {
	encoder := json.NewEncoder(opts.Stdout)
	return &Server{
		logger: opts.Logger,
		stdin:  opts.Stdin,
		stdout: opts.Stdout,
		router: opts.Router,

		encoder: encoder,
		nextID:  1,
		pending: make(map[int64]chan inboundResponse),
	}
}

type inboundResponse struct {
	result  json.RawMessage
	errObj  *protocol.JSONRPCErrorObject
}

type jsonRpcEnvelope struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id"`

	// Present on JSON-RPC requests.
	Method *string `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`

	// Present on JSON-RPC responses.
	Result json.RawMessage `json:"result,omitempty"`
	Error  *protocol.JSONRPCErrorObject `json:"error,omitempty"`
}

// Request sends an outbound JSON-RPC request (daemon -> extension) and blocks
// until a matching response (extension -> daemon) is received.
func (s *Server) Request(method string, params any) (json.RawMessage, error) {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	s.outboundMu.Lock()
	id := s.nextID
	s.nextID++
	s.outboundMu.Unlock()

	ch := make(chan inboundResponse, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()

	req := protocol.JSONRPCRequest[json.RawMessage]{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	s.writeMu.Lock()
	encErr := s.encoder.Encode(req)
	s.writeMu.Unlock()
	if encErr != nil {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
		return nil, encErr
	}

	resp := <-ch
	if resp.errObj != nil {
		return nil, fmt.Errorf("%d: %s", resp.errObj.Code, resp.errObj.Message)
	}
	return resp.result, nil
}

func (s *Server) ApplyDirectives(
	params protocol.HostApplyParams,
) (protocol.HostApplyResult, error) {
	raw, err := s.Request("host.applyDirectives", params)
	if err != nil {
		return protocol.HostApplyResult{}, err
	}
	var out protocol.HostApplyResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return protocol.HostApplyResult{}, err
	}
	return out, nil
}

func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.stdin)

	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	var wg sync.WaitGroup
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var env jsonRpcEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			if s.logger != nil {
				s.logger.Printf("failed to parse JSON-RPC message: %v", err)
			}
			continue
		}

		// Request: JSONRPCRequest has a "method" field.
		if env.Method != nil {
			var req protocol.JSONRPCRequest[json.RawMessage]
			if err := json.Unmarshal(line, &req); err != nil {
				if s.logger != nil {
					s.logger.Printf("failed to decode JSON-RPC request: %v", err)
				}
				s.encodeError(nil, NewParseError())
				continue
			}

			wg.Add(1)
			go func(req protocol.JSONRPCRequest[json.RawMessage]) {
				defer wg.Done()

				result, rpcErr := s.router.Handle(req)
				if rpcErr != nil {
					id := req.ID
					s.encodeError(&id, rpcErr)
					return
				}

				if err := s.validateResult(result); err != nil {
					if s.logger != nil {
						s.logger.Printf("invalid handler result for %s: %v", req.Method, err)
					}
					id := req.ID
					s.encodeError(&id, NewInternalError(err))
					return
				}

				s.encodeSuccess(req.ID, result)
			}(req)
			continue
		}

		// Response: route to pending outbound request (daemon -> extension).
		if env.ID == nil {
			continue
		}

		id := *env.ID
		s.pendingMu.Lock()
		ch := s.pending[id]
		if ch != nil {
			delete(s.pending, id)
		}
		s.pendingMu.Unlock()
		if ch == nil {
			continue
		}

		ch <- inboundResponse{
			result: env.Result,
			errObj: env.Error,
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		wg.Wait()
		return err
	}

	wg.Wait()
	return nil
}

func (s *Server) encodeSuccess(id int64, result any) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.encoder.Encode(NewSuccessResponse(id, result)); err != nil {
		s.logger.Printf("failed to encode success response: %v", err)
	}
}

func (s *Server) encodeError(id *int64, rpcErr *protocol.JSONRPCErrorObject) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.encoder.Encode(NewErrorResponse(id, rpcErr)); err != nil {
		s.logger.Printf("failed to encode error response: %v", err)
	}
}

type resultValidator interface {
	Validate() error
}

func (s *Server) validateResult(result any) error {
	switch v := result.(type) {
	case resultValidator:
		return v.Validate()
	case *resultValidator:
		if v == nil {
			return nil
		}
		return (*v).Validate()
	default:
		return nil
	}
}
