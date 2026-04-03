package app

import (
	"fmt"
	"strings"
)

// handleConfig applies a live config patch without restarting the process.
func (a *App) handleConfig(patch *ConfigPatch) error {
	if patch == nil {
		return a.write(Event{Type: "error", Message: "config patch missing"})
	}

	a.cfgMu.Lock()
	next := a.cfg
	var err error
	next, err = applyConfigPatch(next, patch)
	if err != nil {
		a.cfgMu.Unlock()
		return a.write(Event{Type: "error", Message: err.Error()})
	}
	a.cfg = next
	a.cfgMu.Unlock()

	// Signal the transcribe loop. Best-effort (non-blocking).
	select {
	case a.cfgUpdateCh <- struct{}{}:
	default:
	}

	// Optional ack so clients can observe config changes if desired.
	_ = a.write(Event{Type: "state", State: "config_updated"})
	return nil
}

func applyConfigPatch(base SidecarConfig, patch *ConfigPatch) (SidecarConfig, error) {
	out := base

	if patch.SttModelId != nil {
		v := strings.TrimSpace(*patch.SttModelId)
		if v == "" {
			return SidecarConfig{}, fmt.Errorf("sttModelId must be non-empty")
		}
		out.SttModelId = v
	}

	if patch.SttLanguage != nil {
		lang, code := normalizeLanguageCode(*patch.SttLanguage)
		out.SttLanguage = lang
		out.SttLanguageCode = code
	}

	if patch.VadDebug != nil {
		out.VadDebugEnabled = *patch.VadDebug
	}

	// VAD + segmentation tuning.
	if patch.VadThresholdMultiplier != nil {
		if *patch.VadThresholdMultiplier < 0.1 || *patch.VadThresholdMultiplier > 10.0 {
			return SidecarConfig{}, fmt.Errorf("vadThresholdMultiplier out of range")
		}
		out.VadThresholdMultiplier = *patch.VadThresholdMultiplier
	}
	if patch.VadMinEnergyFloor != nil {
		if *patch.VadMinEnergyFloor < 30 || *patch.VadMinEnergyFloor > 800 {
			return SidecarConfig{}, fmt.Errorf("vadMinEnergyFloor out of range")
		}
		out.VadMinEnergyFloor = *patch.VadMinEnergyFloor
	}
	if patch.VadStartMs != nil {
		if *patch.VadStartMs < 20 || *patch.VadStartMs > 2000 {
			return SidecarConfig{}, fmt.Errorf("vadStartMs out of range")
		}
		out.VadStartMs = *patch.VadStartMs
	}
	if patch.VadEndMs != nil {
		if *patch.VadEndMs < 60 || *patch.VadEndMs > 5000 {
			return SidecarConfig{}, fmt.Errorf("vadEndMs out of range")
		}
		out.VadEndMs = *patch.VadEndMs
	}
	if patch.VadPrerollMs != nil {
		if *patch.VadPrerollMs < 0 || *patch.VadPrerollMs > 1000 {
			return SidecarConfig{}, fmt.Errorf("vadPrerollMs out of range")
		}
		out.VadPrerollMs = *patch.VadPrerollMs
	}

	if patch.SttCommitResponseTimeoutMs != nil {
		if *patch.SttCommitResponseTimeoutMs < 0 || *patch.SttCommitResponseTimeoutMs > 180_000 {
			return SidecarConfig{}, fmt.Errorf(
				"sttCommitResponseTimeoutMs must be within [0, 180000]",
			)
		}
		out.SttCommitResponseTimeoutMs = *patch.SttCommitResponseTimeoutMs
	}

	if patch.StreamMinChunkMs != nil {
		if *patch.StreamMinChunkMs < 50 || *patch.StreamMinChunkMs > 2000 {
			return SidecarConfig{}, fmt.Errorf("streamMinChunkMs out of range")
		}
		out.StreamMinChunkMs = *patch.StreamMinChunkMs
	}
	if patch.StreamMaxChunkMs != nil {
		if *patch.StreamMaxChunkMs < 50 || *patch.StreamMaxChunkMs > 3000 {
			return SidecarConfig{}, fmt.Errorf("streamMaxChunkMs out of range")
		}
		out.StreamMaxChunkMs = *patch.StreamMaxChunkMs
	}
	if patch.StreamMaxUtteranceMs != nil {
		out.StreamMaxUtteranceMs = normalizeStreamMaxUtteranceMS(*patch.StreamMaxUtteranceMs)
	}

	return out, nil
}
