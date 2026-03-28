package app

import (
	"io"
	"log"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/requestcontext"
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
	agentRuntime := agent.New(stub.New())
	editEngine := edit.NewEngine()
	reqProvider := requestcontext.NewProvider(symbols.NewTreeSitterResolver())
	intentHandler := dispatch.NewHandler(editEngine, reqProvider)

	voiceService := transcript.NewService(agentRuntime, intentHandler)

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

	return &App{
		logger: opts.Logger,
		server: server,
	}, nil
}

func (a *App) Run() error {
	a.logger.Println("vocoded starting...")
	return a.server.Run()
}
