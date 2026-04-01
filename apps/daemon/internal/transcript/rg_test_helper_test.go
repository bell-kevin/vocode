package transcript

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// requireRipgrep skips the test when ripgrep cannot be run (no VOCODE_RG_BIN file, no `rg` on PATH).
func requireRipgrep(t *testing.T) {
	t.Helper()
	if p := strings.TrimSpace(os.Getenv("VOCODE_RG_BIN")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return
		}
	}
	if _, err := exec.LookPath("rg"); err == nil {
		return
	}
	t.Skip("ripgrep not available: set VOCODE_RG_BIN to the rg binary or install rg on PATH")
}
