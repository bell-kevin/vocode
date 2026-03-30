package app

import (
	"io"
	"log"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/anthropic"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/openai"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/gather"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/rpc"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript"
)

type Options struct {
	Logger *log.Logger
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type App struct {
	logger *log.Logger
	server *rpc.Server
}

func New(opts Options) (*App, error) {
	agentRuntime := agent.New(selectModelClient(opts.Logger))
	editEngine := edit.NewEngine()
	sym := symbols.NewTreeSitterResolver()
	gatherProvider := gather.NewProvider(sym)
	intentHandler := dispatch.NewHandler(editEngine)
	voiceService := transcript.NewService(agentRuntime, intentHandler, gatherProvider, sym, opts.Logger)

	router := rpc.NewRouter(opts.Logger)
	for _, def := range rpc.BuildHandlers(voiceService) {
		router.Register(def.Method, def.Handler)
	}

	server := rpc.NewServer(rpc.ServerOptions{
		Logger: opts.Logger,
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Router: router,
	})

	voiceService.SetHostApplyClient(server)

	return &App{
		logger: opts.Logger,
		server: server,
	}, nil
}

func selectModelClient(logger *log.Logger) agent.ModelClient {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("VOCODE_AGENT_PROVIDER")))
	switch provider {
	case "", "stub":
		return stub.New()
	case "openai":
		c, err := openai.NewFromEnv()
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: OpenAI unavailable (%v); using stub model client", err)
			}
			return stub.New()
		}
		if logger != nil {
			logger.Printf("vocode agent: using OpenAI model client")
		}
		return c
	case "anthropic":
		c, err := anthropic.NewFromEnv()
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: Anthropic unavailable (%v); using stub model client", err)
			}
			return stub.New()
		}
		if logger != nil {
			logger.Printf("vocode agent: using Anthropic model client")
		}
		return c
	default:
		if logger != nil {
			logger.Printf("vocode agent: unknown VOCODE_AGENT_PROVIDER %q; using stub model client", provider)
		}
		return stub.New()
	}
}

func (a *App) Run() error {
	a.logger.Println("vocoded starting...")
	return a.server.Run()
}
