package search

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RgBinary returns VOCODE_RG_BIN if set, else "rg".
func RgBinary() string {
	if p := strings.TrimSpace(os.Getenv("VOCODE_RG_BIN")); p != "" {
		return p
	}
	return "rg"
}

var rgLineRe = regexp.MustCompile(`^(.*):(\d+):(\d+):(.*)$`)

// FixedStringSearch runs ripgrep with --fixed-strings under root and returns up to maxHits matches.
// Exit code 1 (no matches) yields nil, nil. maxHits <= 0 defaults to 20.
func FixedStringSearch(root, query string, maxHits int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if maxHits <= 0 {
		maxHits = 20
	}

	cmd := exec.Command(RgBinary(), "--column", "-n", "--fixed-strings", query, root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}

	out := make([]Hit, 0, maxHits)
	sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := rgLineRe.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		path := filepath.Clean(strings.TrimSpace(m[1]))
		ln0 := atoiSafe(m[2])
		col0 := atoiSafe(m[3])
		if ln0 <= 0 || col0 <= 0 {
			continue
		}
		out = append(out, Hit{
			Path:    path,
			Line0:   ln0 - 1,
			Char0:   col0 - 1,
			Len:     len(query),
			Preview: strings.TrimSpace(m[4]),
		})
		if len(out) >= maxHits {
			break
		}
	}
	return out, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
