package app

import (
	"io"
	"log"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/rpc"
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
	agentService := agent.NewService()
	editService := edits.NewService(agentService)

	router := rpc.NewRouter(opts.Logger)
	for _, def := range rpc.BuildHandlers(editService) {
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
