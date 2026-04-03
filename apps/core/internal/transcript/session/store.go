package session

import (
	"fmt"
	"strings"
	"sync"
	"time"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// SearchHit is a single ripgrep match surfaced in the selection panel.
type SearchHit struct {
	Path      string
	Line      int
	Character int
	Preview   string
	Len       int
}

// VoiceSession is core’s per-voice transcript state.
// It is persisted when `contextSessionId` is non-empty and stored ephemerally per-RPC otherwise.
type VoiceSession struct {
	// Placeholder for later gathered context used by the executor.
	// Implemented as `any` for now to avoid forcing porting symbols/excerpts before
	// flow orchestration is proven.
	Gathered any

	PendingDirectiveApply *DirectiveApplyBatch

	SearchResults     []SearchHit
	ActiveSearchIndex int

	BasePhase BasePhase
	Clarify   *ClarifyOverlay

	FileSelectionPaths []string
	FileSelectionIsDir []bool
	FileSelectionIndex int
	FileSelectionFocus string
}

// VoiceSessionStore retains VoiceSession between voice.transcript calls.
type VoiceSessionStore struct {
	mu          sync.Mutex
	maxSessions int
	data        map[string]voiceSessionEntry
}

type voiceSessionEntry struct {
	session VoiceSession
	lastPut time.Time
}

func NewVoiceSessionStore() *VoiceSessionStore {
	return &VoiceSessionStore{
		maxSessions: 256,
		data:        make(map[string]voiceSessionEntry),
	}
}

func CloneVoiceSession(v VoiceSession) VoiceSession {
	var pending *DirectiveApplyBatch
	if v.PendingDirectiveApply != nil {
		p := *v.PendingDirectiveApply
		pending = &p
	}

	out := VoiceSession{
		Gathered:              v.Gathered,
		PendingDirectiveApply: pending,
		SearchResults:         append([]SearchHit(nil), v.SearchResults...),
		ActiveSearchIndex:     v.ActiveSearchIndex,
		BasePhase:             v.BasePhase,
		Clarify:               cloneClarifyOverlay(v.Clarify),
		FileSelectionPaths:    append([]string(nil), v.FileSelectionPaths...),
		FileSelectionIsDir:    append([]bool(nil), v.FileSelectionIsDir...),
		FileSelectionIndex:    v.FileSelectionIndex,
		FileSelectionFocus:    v.FileSelectionFocus,
	}

	// Ensure slices are nil when empty to keep behavior predictable.
	if len(out.SearchResults) == 0 {
		out.SearchResults = nil
	}
	if len(out.FileSelectionPaths) == 0 {
		out.FileSelectionPaths = nil
	}
	if len(out.FileSelectionIsDir) == 0 {
		out.FileSelectionIsDir = nil
	}
	return out
}

func cloneClarifyOverlay(ov *ClarifyOverlay) *ClarifyOverlay {
	if ov == nil {
		return nil
	}
	tmp := *ov
	return &tmp
}

// FileFocusIsDir reports whether the focused file-selection row is a directory (wire isDirectory).
func (vs *VoiceSession) FileFocusIsDir() bool {
	if vs == nil || len(vs.FileSelectionPaths) == 0 {
		return false
	}
	i := vs.FileSelectionIndex
	if i < 0 || i >= len(vs.FileSelectionPaths) {
		return false
	}
	if i < len(vs.FileSelectionIsDir) {
		return vs.FileSelectionIsDir[i]
	}
	return false
}

// Get returns session state, or empty if unknown, blank key, or idle evicted.
func (s *VoiceSessionStore) Get(key string, idleReset time.Duration) VoiceSession {
	key = strings.TrimSpace(key)
	if key == "" || s == nil {
		return VoiceSession{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ent, ok := s.data[key]
	if !ok {
		return VoiceSession{}
	}

	if idleReset > 0 && !ent.lastPut.IsZero() && time.Since(ent.lastPut) > idleReset {
		delete(s.data, key)
		return VoiceSession{}
	}

	return CloneVoiceSession(ent.session)
}

// Put replaces session state and refreshes last activity time.
func (s *VoiceSessionStore) Put(key string, session VoiceSession) {
	key = strings.TrimSpace(key)
	if key == "" || s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string]voiceSessionEntry)
	}

	max := s.maxSessions
	if max <= 0 {
		max = 256
	}
	for len(s.data) >= max {
		for k := range s.data {
			delete(s.data, k)
			break
		}
	}

	s.data[key] = voiceSessionEntry{session: session, lastPut: time.Now()}
}

// DirectiveApplyBatch is one batch of directives the core daemon returned.
// The host must apply them and respond with per-item statuses.
type DirectiveApplyBatch struct {
	ID            string
	NumDirectives int
}

const (
	ApplyItemStatusOK      = "ok"
	ApplyItemStatusFailed  = "failed"
	ApplyItemStatusSkipped = "skipped"
)

func (b *DirectiveApplyBatch) ConsumeHostApplyReport(
	reportBatchID string,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
) error {
	if b == nil {
		return fmt.Errorf("directive apply batch: nil batch")
	}
	if strings.TrimSpace(reportBatchID) != b.ID {
		return fmt.Errorf("directive apply batch: applyBatchId mismatch")
	}
	if len(items) != b.NumDirectives {
		return fmt.Errorf("directive apply batch: apply items length mismatch")
	}
	for _, it := range items {
		status := strings.TrimSpace(it.Status)
		switch status {
		case ApplyItemStatusOK, ApplyItemStatusSkipped:
			// valid
		case ApplyItemStatusFailed:
			msg := strings.TrimSpace(it.Message)
			if msg == "" {
				msg = "host apply failed"
			}
			return fmt.Errorf("%s", msg)
		default:
			return fmt.Errorf("directive apply batch: unknown status %q", it.Status)
		}
	}
	return nil
}
