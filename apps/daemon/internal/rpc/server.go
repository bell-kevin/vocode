package rpc

import (
	"bufio"
	"encoding/json"
	"io"
	"log"

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
}

func NewServer(opts ServerOptions) *Server {
	return &Server{
		logger: opts.Logger,
		stdin:  opts.Stdin,
		stdout: opts.Stdout,
		router: opts.Router,
	}
}

func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.stdin)

	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	encoder := json.NewEncoder(s.stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req protocol.JSONRPCRequest[json.RawMessage]
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.Printf("failed to parse request: %v", err)
			s.encodeError(encoder, nil, NewParseError())
			continue
		}

		result, rpcErr := s.router.Handle(req)
		if rpcErr != nil {
			id := req.ID
			s.encodeError(encoder, &id, rpcErr)
			continue
		}

		if err := s.validateResult(result); err != nil {
			s.logger.Printf("invalid handler result for %s: %v", req.Method, err)
			id := req.ID
			s.encodeError(encoder, &id, NewInternalError(err))
			continue
		}

		s.encodeSuccess(encoder, req.ID, result)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (s *Server) encodeSuccess(
	encoder *json.Encoder,
	id int64,
	result any,
) {
	if err := encoder.Encode(NewSuccessResponse(id, result)); err != nil {
		s.logger.Printf("failed to encode success response: %v", err)
	}
}

func (s *Server) encodeError(
	encoder *json.Encoder,
	id *int64,
	rpcErr *protocol.JSONRPCErrorObject,
) {
	if err := encoder.Encode(NewErrorResponse(id, rpcErr)); err != nil {
		s.logger.Printf("failed to encode error response: %v", err)
	}
}

func (s *Server) validateResult(result any) error {
	switch v := result.(type) {
	case protocol.EditApplyResult:
		return v.Validate()
	case *protocol.EditApplyResult:
		if v == nil {
			return nil
		}
		return v.Validate()
	case protocol.VoiceTranscriptResult:
		return v.Validate()
	case *protocol.VoiceTranscriptResult:
		if v == nil {
			return nil
		}
		return v.Validate()
	default:
		return nil
	}
}
