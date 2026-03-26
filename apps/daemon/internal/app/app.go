package app

import (
	"io"
	"log"

	"vocoding.net/vocode/v2/apps/daemon/internal/actionplan/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/commandexec"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/rpc"
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
	editService := edits.NewService()
	commandService := commandexec.NewService()
	dispatcher := dispatch.NewDispatcher(editService, commandService)

	voiceService := transcript.NewService(agentRuntime, dispatcher)

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
