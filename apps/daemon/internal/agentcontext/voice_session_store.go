package agentcontext

import (
	"strings"
	"sync"
	"time"
)

// VoiceSession is all daemon state keyed by params.contextSessionId for one voice
// transcript stream: accumulated gathered context plus at most one open directive-apply batch
// awaiting host report (see [DirectiveApplyBatch]), and cumulative host apply outcomes for the model.
type VoiceSession struct {
	Gathered              Gathered
	PendingDirectiveApply *DirectiveApplyBatch
	IntentApplyHistory    []IntentApplyRecord
	NextApplyBatchOrdinal int
}

// VoiceSessionStore retains [VoiceSession] between voice.transcript RPCs.
// Get drops stored state when idleReset > 0 and nothing was saved longer than idleReset (since last Put).
type VoiceSessionStore struct {
	mu sync.Mutex

	maxSessions int
	data        map[string]voiceSessionEntry
}

type voiceSessionEntry struct {
	session VoiceSession
	lastPut time.Time
}

// NewVoiceSessionStore returns a store with a default session cap.
func NewVoiceSessionStore() *VoiceSessionStore {
	return &VoiceSessionStore{
		maxSessions: 256,
		data:        make(map[string]voiceSessionEntry),
	}
}

// Get returns session state, or empty if unknown, blank key, or idle elapsed since last Put.
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
	return cloneVoiceSession(ent.session)
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

func cloneVoiceSession(v VoiceSession) VoiceSession {
	return CloneVoiceSession(v)
}
