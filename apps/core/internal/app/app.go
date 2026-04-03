package app

import (
	"io"
	"log"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/agent/anthropic"
	"vocoding.net/vocode/v2/apps/core/internal/agent/openai"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/rpc"
	"vocoding.net/vocode/v2/apps/core/internal/transcript"
)

type App struct {
	logger *log.Logger
	server *rpc.Server
}

// New constructs the core daemon runtime.
func New(opts Options) (*App, error) {
	var stdin io.Reader = opts.Stdin
	var stdout io.Writer = opts.Stdout

	var editModel agent.ModelClient
	if c, err := openai.NewFromEnv(); err == nil {
		editModel = c
	} else if c2, err2 := anthropic.NewFromEnv(); err2 == nil {
		editModel = c2
	}
	voiceService := transcript.NewService(selectFlowRouter(opts.Logger), editModel)

	r := rpc.NewRouter(opts.Logger)
	for _, def := range rpc.BuildHandlers(voiceService) {
		r.Register(def.Method, def.Handler)
	}

	server := rpc.NewServer(rpc.ServerOptions{
		Logger: opts.Logger,
		Stdin:  stdin,
		Stdout: stdout,
		Router: r,
	})

	voiceService.SetHostApplyClient(server)

	return &App{
		logger: opts.Logger,
		server: server,
	}, nil
}

func selectFlowRouter(logger *log.Logger) *router.FlowRouter {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("VOCODE_AGENT_PROVIDER")))
	switch provider {
	case "", "stub":
		return router.NewFlowRouter(nil)
	case "openai":
		c, err := openai.NewFromEnv()
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: OpenAI unavailable (%v); using heuristic flow router", err)
			}
			return router.NewFlowRouter(nil)
		}
		if logger != nil {
			logger.Printf("vocode agent: using OpenAI model for flow routing")
		}
		return router.NewFlowRouter(c)
	case "anthropic":
		c, err := anthropic.NewFromEnv()
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: Anthropic unavailable (%v); using heuristic flow router", err)
			}
			return router.NewFlowRouter(nil)
		}
		if logger != nil {
			logger.Printf("vocode agent: using Anthropic model for flow routing")
		}
		return router.NewFlowRouter(c)
	default:
		if logger != nil {
			logger.Printf("vocode agent: unknown VOCODE_AGENT_PROVIDER %q; using heuristic flow router", provider)
		}
		return router.NewFlowRouter(nil)
	}
}

func (a *App) Run() error {
	if a.logger != nil {
		a.logger.Println("vocode-cored starting...")
	}
	return a.server.Run()
}
