package workspaceselectflow

import (
	"strings"
	"testing"
)

func TestStackPromptAddenda_noBuiltinByDefault(t *testing.T) {
	t.Parallel()
	if s := StackPromptAddenda(nil, ""); s != "" {
		t.Fatalf("want empty when no skills and no addendum, got %q", s)
	}
	if s := StackPromptAddenda([]string{}, ""); s != "" {
		t.Fatalf("want empty for empty skill list, got %q", s)
	}
}

func TestStackPromptAddenda_reactNativeExpoSkill(t *testing.T) {
	t.Parallel()
	s := StackPromptAddenda([]string{"react-native-expo"}, "")
	if !strings.Contains(s, "React Native") || !strings.Contains(s, "Expo") {
		t.Fatalf("expected RN/Expo block, got %q", s)
	}
}

func TestStackPromptAddenda_promptAddendumOnly(t *testing.T) {
	t.Parallel()
	s := StackPromptAddenda(nil, "Use pnpm.")
	if !strings.Contains(s, "Project (.vocode)") || !strings.Contains(s, "Use pnpm.") {
		t.Fatalf("got %q", s)
	}
}
