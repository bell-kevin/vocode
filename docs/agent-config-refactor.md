### Agent config refactor (provider/model/base URL)

This document describes a future refactor to stop using environment variables for the agent provider/model/base URL selection (except for API keys), and instead pass an explicit configuration from the extension to the daemon, similar to how `daemonConfig` works for planner caps.

#### 1. Introduce explicit agent config in the daemon

- Add a new type in the daemon agent package (e.g. `apps/daemon/internal/agent/config.go`):

```go
type AgentProvider string

const (
	AgentProviderStub      AgentProvider = "stub"
	AgentProviderOpenAI    AgentProvider = "openai"
	AgentProviderAnthropic AgentProvider = "anthropic"
)

type AgentConfig struct {
	Provider         AgentProvider
	OpenAIModel      string
	OpenAIBaseURL    string
	AnthropicModel   string
	AnthropicBaseURL string
}
```

- Add a constructor that takes this config instead of reading env:

```go
// NewWithConfig constructs an Agent from an explicit config.
func NewWithConfig(cfg AgentConfig, logger *log.Logger) *Agent {
	var model ModelClient
	switch cfg.Provider {
	case AgentProviderOpenAI:
		c, err := openai.NewWithConfig(openai.Config{
			APIKey:  os.Getenv("OPENAI_API_KEY"), // API key still from env
			Model:   cfg.OpenAIModel,
			BaseURL: cfg.OpenAIBaseURL,
		})
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: OpenAI unavailable (%v); using stub model client", err)
			}
			model = stub.New()
		} else {
			model = c
		}
	case AgentProviderAnthropic:
		c, err := anthropic.NewWithConfig(anthropic.Config{
			APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
			Model:   cfg.AnthropicModel,
			BaseURL: cfg.AnthropicBaseURL,
		})
		if err != nil {
			if logger != nil {
				logger.Printf("vocode agent: Anthropic unavailable (%v); using stub model client", err)
			}
			model = stub.New()
		} else {
			model = c
		}
	default:
		model = stub.New()
	}
	return &Agent{model: model}
}
```

- Keep the existing `New` / `selectModelClient` / `NewFromEnv` paths as thin shims for CLI and legacy use, but have them build an `AgentConfig` from env and call `NewWithConfig`.

#### 2. Add config-aware constructors in model clients

- In `apps/daemon/internal/agent/openai/client.go` introduce:

```go
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewWithConfig(cfg Config) (*Client, error) {
	// validate + default BaseURL / Model without reading env
}

func NewFromEnv() (*Client, error) {
	return NewWithConfig(Config{
		APIKey:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		BaseURL: strings.TrimSpace(os.Getenv("VOCODE_OPENAI_BASE_URL")),
		Model:   strings.TrimSpace(os.Getenv("VOCODE_OPENAI_MODEL")),
	})
}
```

- Mirror the same pattern for the Anthropic client.

#### 3. Change daemon bootstrap to accept explicit config

- In `apps/daemon/internal/app/app.go`:
  - Replace direct calls to `selectModelClient(logger)` with a path that can accept an `AgentConfig`.
  - For now, build `AgentConfig` from env inside `selectModelClient` and delegate to `agent.NewWithConfig`, so behaviour stays identical while the extension is still passing provider/model/base URL via env.
  - Later, add a way to pass `AgentConfig` into `New` without env (e.g. command-line flags or a small JSON config file).

#### 4. Update the extension to stop using env for provider/model/base URL

- In `apps/vscode-extension/src/config/spawn-env.ts`:
  - Remove the `CONFIG_TO_ENV` bindings for:
    - `daemonAgentProvider` → `VOCODE_AGENT_PROVIDER`
    - `daemonOpenaiModel` → `VOCODE_OPENAI_MODEL`
    - `daemonOpenaiBaseUrl` → `VOCODE_OPENAI_BASE_URL`
    - `daemonAnthropicModel` → `VOCODE_ANTHROPIC_MODEL`
    - `daemonAnthropicBaseUrl` → `VOCODE_ANTHROPIC_BASE_URL`
  - Keep API keys in env (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `ELEVENLABS_API_KEY`) exactly as today.

- When spawning the daemon process, pass a small JSON config blob or command-line flags that mirror `AgentConfig`:
  - Values come from `vscode.workspace.getConfiguration("vocode")`:
    - `daemonAgentProvider`
    - `daemonOpenaiModel`
    - `daemonOpenaiBaseUrl`
    - `daemonAnthropicModel`
    - `daemonAnthropicBaseUrl`

#### 5. Tests and validation

- Add tests around:
  - `openai.NewWithConfig` and `anthropic.NewWithConfig` (no env reads).
  - `agent.NewWithConfig`:
    - Selects OpenAI/Anthropic/stub based on `Provider`.
    - Falls back to stub when a provider-specific config is incomplete or the client constructor errors.
  - End-to-end executor tests that construct an `Agent` via `NewWithConfig` and run a transcript without relying on provider/model/base URL env vars.

- Once the extension is updated and tests are green:
  - Consider marking env-based provider/model/base URL selection as deprecated in comments.
  - Optionally remove those env reads from the main daemon path, keeping them only for direct `go run`/CLI workflows if needed.

