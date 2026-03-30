package intents

import (
	"encoding/json"
	"fmt"
	"strings"
)

// --- Control intents (agent loop only; no protocol directives) ---

type ControlIntentKind string

const (
	ControlIntentKindDone           ControlIntentKind = "done"
	ControlIntentKindRequestContext ControlIntentKind = "request_context"
)

// ControlIntent is a non-executable agent step: stop the loop or enrich gathered context.
type ControlIntent struct {
	Kind           ControlIntentKind     `json:"kind"`
	RequestContext *RequestContextIntent `json:"requestContext,omitempty"`
	Done           *DoneIntent           `json:"done,omitempty"`
}

// --- Executable intents (produce protocol directives for the host) ---

type ExecutableIntentKind string

const (
	ExecutableIntentKindEdit     ExecutableIntentKind = "edit"
	ExecutableIntentKindCommand  ExecutableIntentKind = "command"
	ExecutableIntentKindNavigate ExecutableIntentKind = "navigate"
	ExecutableIntentKindUndo     ExecutableIntentKind = "undo"
)

// ExecutableIntent is a host-observable step: maps to at most one directive kind.
type ExecutableIntent struct {
	Kind     ExecutableIntentKind `json:"kind"`
	Edit     *EditIntent          `json:"edit,omitempty"`
	Command  *CommandIntent       `json:"command,omitempty"`
	Navigate *NavigationIntent    `json:"navigate,omitempty"`
	Undo     *UndoIntent          `json:"undo,omitempty"`
}

// Intent is exactly one of [ControlIntent] or [ExecutableIntent] (union).
// JSON is a single object with top-level "kind" (and payload fields), e.g.
// {"kind":"done"}, {"kind":"done","done":{"summary":"..."}}, {"kind":"request_context","requestContext":{...}}, {"kind":"edit","edit":{...}}.
type Intent struct {
	Control    *ControlIntent    `json:"-"`
	Executable *ExecutableIntent `json:"-"`
}

// ControlDone returns an intent that stops the agent loop.
func ControlDone() Intent {
	return Intent{Control: &ControlIntent{Kind: ControlIntentKindDone}}
}

// ControlDoneSummary returns a done intent with optional human-readable summary for the host UI.
func ControlDoneSummary(summary string) Intent {
	return Intent{Control: &ControlIntent{Kind: ControlIntentKindDone, Done: &DoneIntent{Summary: summary}}}
}

// ControlRequestContext returns an intent that enriches gathered context (symbols, excerpts, notes).
func ControlRequestContext(req *RequestContextIntent) Intent {
	return Intent{Control: &ControlIntent{Kind: ControlIntentKindRequestContext, RequestContext: req}}
}

// FromExecutable wraps an executable intent as a union value.
func FromExecutable(e ExecutableIntent) Intent {
	return Intent{Executable: &e}
}

// Validate checks the union invariant and payload constraints.
func (i Intent) Validate() error {
	return validateIntent(i)
}

// Summary returns a short intent label (e.g. "done", "request_context", "edit").
func (i Intent) Summary() string {
	if i.Control != nil {
		return string(i.Control.Kind)
	}
	if i.Executable != nil {
		return string(i.Executable.Kind)
	}
	return ""
}

func validateIntent(i Intent) error {
	if i.Control != nil && i.Executable != nil {
		return fmt.Errorf("intent: both control and executable are set")
	}
	if i.Control == nil && i.Executable == nil {
		return fmt.Errorf("intent: empty")
	}
	if i.Control != nil {
		return validateControlIntent(*i.Control)
	}
	return validateExecutableIntent(*i.Executable)
}

func validateControlIntent(c ControlIntent) error {
	switch c.Kind {
	case ControlIntentKindDone:
		if c.RequestContext != nil {
			return fmt.Errorf("intent: kind %q must not set requestContext", c.Kind)
		}
		return validateDoneIntent(c.Done)
	case ControlIntentKindRequestContext:
		if c.RequestContext == nil {
			return fmt.Errorf("intent: kind %q requires requestContext", c.Kind)
		}
		switch c.RequestContext.Kind {
		case RequestContextKindSymbols, RequestContextKindFileExcerpt, RequestContextKindUsages:
			return nil
		default:
			return fmt.Errorf("intent: unknown requestContext kind %q", c.RequestContext.Kind)
		}
	default:
		return fmt.Errorf("intent: unknown control kind %q", c.Kind)
	}
}

func validateExecutableIntent(e ExecutableIntent) error {
	switch e.Kind {
	case ExecutableIntentKindEdit:
		if e.Edit == nil {
			return fmt.Errorf("intent: kind %q requires edit", e.Kind)
		}
		return ValidateEditIntent(*e.Edit)
	case ExecutableIntentKindCommand:
		if e.Command == nil {
			return fmt.Errorf("intent: kind %q requires command", e.Kind)
		}
		if strings.TrimSpace(e.Command.Command) == "" {
			return fmt.Errorf("intent: command.command is empty")
		}
		return nil
	case ExecutableIntentKindNavigate:
		if e.Navigate == nil {
			return fmt.Errorf("intent: kind %q requires navigate", e.Kind)
		}
		return ValidateNavigationIntent(*e.Navigate)
	case ExecutableIntentKindUndo:
		if e.Undo == nil {
			return fmt.Errorf("intent: kind %q requires undo", e.Kind)
		}
		return ValidateUndoIntent(*e.Undo)
	default:
		return fmt.Errorf("intent: unknown executable kind %q", e.Kind)
	}
}

// UnmarshalJSON decodes the flat wire shape into the union.
func (i *Intent) UnmarshalJSON(data []byte) error {
	*i = Intent{}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("intent: %w", err)
	}
	kindRaw, ok := raw["kind"]
	if !ok {
		return fmt.Errorf("intent: missing kind")
	}
	var kind string
	if err := json.Unmarshal(kindRaw, &kind); err != nil {
		return fmt.Errorf("intent: kind: %w", err)
	}
	switch kind {
	case string(ControlIntentKindDone):
		var d *DoneIntent
		if rawDone, ok := raw["done"]; ok && len(rawDone) > 0 && string(rawDone) != "null" {
			d = new(DoneIntent)
			if err := json.Unmarshal(rawDone, d); err != nil {
				return fmt.Errorf("intent: done: %w", err)
			}
		}
		i.Control = &ControlIntent{Kind: ControlIntentKindDone, Done: d}
	case string(ControlIntentKindRequestContext):
		rc := new(RequestContextIntent)
		if err := json.Unmarshal(raw["requestContext"], rc); err != nil {
			return fmt.Errorf("intent: requestContext: %w", err)
		}
		i.Control = &ControlIntent{Kind: ControlIntentKindRequestContext, RequestContext: rc}
	case string(ExecutableIntentKindEdit):
		ex := ExecutableIntent{Kind: ExecutableIntentKindEdit}
		ex.Edit = new(EditIntent)
		if err := json.Unmarshal(raw["edit"], ex.Edit); err != nil {
			return fmt.Errorf("intent: edit: %w", err)
		}
		i.Executable = &ex
	case string(ExecutableIntentKindCommand):
		ex := ExecutableIntent{Kind: ExecutableIntentKindCommand}
		ex.Command = new(CommandIntent)
		if err := json.Unmarshal(raw["command"], ex.Command); err != nil {
			return fmt.Errorf("intent: command: %w", err)
		}
		i.Executable = &ex
	case string(ExecutableIntentKindNavigate):
		ex := ExecutableIntent{Kind: ExecutableIntentKindNavigate}
		ex.Navigate = new(NavigationIntent)
		if err := json.Unmarshal(raw["navigate"], ex.Navigate); err != nil {
			return fmt.Errorf("intent: navigate: %w", err)
		}
		i.Executable = &ex
	case string(ExecutableIntentKindUndo):
		ex := ExecutableIntent{Kind: ExecutableIntentKindUndo}
		ex.Undo = new(UndoIntent)
		if err := json.Unmarshal(raw["undo"], ex.Undo); err != nil {
			return fmt.Errorf("intent: undo: %w", err)
		}
		i.Executable = &ex
	default:
		return fmt.Errorf("intent: unknown kind %q", kind)
	}
	if err := validateIntent(*i); err != nil {
		return err
	}
	return nil
}

// MarshalJSON encodes the flat wire shape (top-level kind + payload).
func (i Intent) MarshalJSON() ([]byte, error) {
	if err := validateIntent(i); err != nil {
		return nil, err
	}
	if c := i.Control; c != nil {
		switch c.Kind {
		case ControlIntentKindDone:
			if c.Done != nil {
				return json.Marshal(struct {
					Kind string      `json:"kind"`
					Done *DoneIntent `json:"done"`
				}{Kind: string(c.Kind), Done: c.Done})
			}
			return json.Marshal(struct {
				Kind string `json:"kind"`
			}{Kind: string(c.Kind)})
		case ControlIntentKindRequestContext:
			return json.Marshal(struct {
				Kind           string                `json:"kind"`
				RequestContext *RequestContextIntent `json:"requestContext"`
			}{Kind: string(c.Kind), RequestContext: c.RequestContext})
		default:
			return nil, fmt.Errorf("intent: marshal unsupported control kind %q", c.Kind)
		}
	}
	e := i.Executable
	out := struct {
		Kind     string            `json:"kind"`
		Edit     *EditIntent       `json:"edit,omitempty"`
		Command  *CommandIntent    `json:"command,omitempty"`
		Navigate *NavigationIntent `json:"navigate,omitempty"`
		Undo     *UndoIntent       `json:"undo,omitempty"`
	}{Kind: string(e.Kind)}
	switch e.Kind {
	case ExecutableIntentKindEdit:
		out.Edit = e.Edit
	case ExecutableIntentKindCommand:
		out.Command = e.Command
	case ExecutableIntentKindNavigate:
		out.Navigate = e.Navigate
	case ExecutableIntentKindUndo:
		out.Undo = e.Undo
	default:
		return nil, fmt.Errorf("intent: marshal unsupported executable kind %q", e.Kind)
	}
	return json.Marshal(out)
}
