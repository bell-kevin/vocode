import { useState } from "react";

import type { VocodePanelConfig } from "./panel-config-types";
import {
  AdvancedDisclosure,
  LanguageSelectRow,
  OptionalZeroSliderRow,
  SLIDER_SPECS,
  SliderRow,
  SttModelChoiceRow,
} from "./settings-widgets";
import { getVsCodeApi } from "./vscode-api";

export type PanelConfig = VocodePanelConfig;

function patchConfig(patch: Partial<VocodePanelConfig>) {
  getVsCodeApi()?.postMessage({ type: "setPanelConfig", patch });
}

function formatMs(ms: number): string {
  if (ms >= 60_000) {
    const m = ms / 60_000;
    return Number.isInteger(m) ? `${m} min` : `${m.toFixed(1)} min`;
  }
  return `${ms} ms`;
}

function ToggleRow(props: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (next: boolean) => void;
}) {
  const { id, label, description, checked, disabled, onChange } = props;
  return (
    <label className="settings-row" htmlFor={id}>
      <div className="settings-row-text">
        <span className="settings-row-label">{label}</span>
        <span className="settings-row-desc">{description}</span>
      </div>
      <input
        id={id}
        type="checkbox"
        className="settings-toggle"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
    </label>
  );
}

export function SettingsPanel(props: { config: PanelConfig | null }) {
  const { config } = props;
  const api = getVsCodeApi();
  const disabled = !api || config === null;
  const [keyDraft, setKeyDraft] = useState("");

  return (
    <div className="settings-root">
      {config === null ? (
        <p className="settings-loading">Loading options…</p>
      ) : null}
      {config && !config.elevenLabsApiKeyConfigured ? (
        <div className="settings-banner" role="status">
          Add your ElevenLabs API key below to use voice. Keys are stored in VS
          Code secret storage (not settings.json).
        </div>
      ) : null}

      <p className="settings-intro-short">
        Vocode configuration. Changes apply automatically.
      </p>

      {config ? (
        <section className="settings-section">
          <h2 className="settings-section-title">API key</h2>
          <div className="settings-api-row">
            <input
              type="password"
              className="settings-input settings-input-grow"
              autoComplete="off"
              placeholder={
                config.elevenLabsApiKeyConfigured
                  ? "•••••••• (enter to replace)"
                  : "ElevenLabs API key"
              }
              disabled={disabled}
              value={keyDraft}
              onChange={(e) => setKeyDraft(e.target.value)}
            />
            <button
              type="button"
              className="settings-btn"
              disabled={disabled || !keyDraft.trim()}
              onClick={() => {
                api?.postMessage({
                  type: "setElevenLabsApiKey",
                  value: keyDraft.trim(),
                });
                setKeyDraft("");
              }}
            >
              Save key
            </button>
            <button
              type="button"
              className="settings-btn settings-btn-ghost"
              disabled={disabled || !config.elevenLabsApiKeyConfigured}
              onClick={() =>
                api?.postMessage({ type: "setElevenLabsApiKey", value: "" })
              }
            >
              Remove
            </button>
          </div>
          {config.elevenLabsApiKeyConfigured ? (
            <p className="settings-subtle">A key is saved on this machine.</p>
          ) : null}
        </section>
      ) : null}

      {config ? (
        <section className="settings-section">
          <h2 className="settings-section-title">Speech (ElevenLabs)</h2>
          <p className="settings-subtle">
            Workspace STT keywords: add a <code>.vocode</code> JSON file at a
            folder root with <code>sttKeywords</code> (string array). Command
            Palette: <strong>Vocode: Create Workspace .vocode File</strong>,
            then apply &amp; restart.
          </p>
          <div className="settings-field-stack">
            <LanguageSelectRow
              value={config.elevenLabsSttLanguage}
              disabled={disabled}
              onCommit={(v) => patchConfig({ elevenLabsSttLanguage: v })}
            />
            <SttModelChoiceRow
              value={config.elevenLabsSttModelId}
              disabled={disabled}
              onCommit={(v) => patchConfig({ elevenLabsSttModelId: v })}
            />
            <OptionalZeroSliderRow
              label="Commit response timeout"
              hint="When off, wait indefinitely for a response from the speech-to-text server"
              toggleLabel="Limit how long to wait for speech-to-text server response"
              value={config.voiceSttCommitResponseTimeoutMs}
              spec={SLIDER_SPECS.voiceSttCommitResponseTimeoutMs}
              disabled={disabled}
              enableDefault={5000}
              onCommit={(n) =>
                patchConfig({ voiceSttCommitResponseTimeoutMs: n })
              }
              formatDisplay={formatMs}
              offSummary="Unlimited"
            />
          </div>
        </section>
      ) : null}

      {config ? (
        <AdvancedDisclosure title="Advanced">
          <div className="settings-advanced-inner">
            <h3 className="settings-subhead">VAD</h3>
            <div className="settings-field-stack">
              <SliderRow
                label="Threshold multiplier"
                value={config.voiceVadThresholdMultiplier}
                spec={SLIDER_SPECS.voiceVadThresholdMultiplier}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ voiceVadThresholdMultiplier: n })
                }
              />
              <SliderRow
                label="Min energy floor"
                value={config.voiceVadMinEnergyFloor}
                spec={SLIDER_SPECS.voiceVadMinEnergyFloor}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadMinEnergyFloor: n })}
              />
              <SliderRow
                label="Start (ms)"
                value={config.voiceVadStartMs}
                spec={SLIDER_SPECS.voiceVadStartMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadStartMs: n })}
              />
              <SliderRow
                label="End (ms)"
                value={config.voiceVadEndMs}
                spec={SLIDER_SPECS.voiceVadEndMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadEndMs: n })}
              />
              <SliderRow
                label="Preroll (ms)"
                value={config.voiceVadPrerollMs}
                spec={SLIDER_SPECS.voiceVadPrerollMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadPrerollMs: n })}
              />
            </div>
            <h3 className="settings-subhead">Stream</h3>
            <div className="settings-field-stack">
              <SliderRow
                label="Min chunk (ms)"
                value={config.voiceStreamMinChunkMs}
                spec={SLIDER_SPECS.voiceStreamMinChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMinChunkMs: n })}
              />
              <SliderRow
                label="Max chunk (ms)"
                value={config.voiceStreamMaxChunkMs}
                spec={SLIDER_SPECS.voiceStreamMaxChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMaxChunkMs: n })}
              />
              <OptionalZeroSliderRow
                label="Max utterance (ms)"
                hint="When off, commits only on silence or end of stream. When on, force a commit at most every N ms during continuous speech."
                toggleLabel="Cap utterance length (periodic commit)"
                value={config.voiceStreamMaxUtteranceMs}
                spec={SLIDER_SPECS.voiceStreamMaxUtteranceMs}
                disabled={disabled}
                enableDefault={8000}
                onCommit={(n) => patchConfig({ voiceStreamMaxUtteranceMs: n })}
                formatDisplay={formatMs}
                offSummary="Unlimited"
              />
            </div>
            <h3 className="settings-subhead">Daemon (voice transcript)</h3>
            <div className="settings-field-stack">
              <SliderRow
                label="Queue size"
                value={config.daemonVoiceTranscriptQueueSize}
                spec={SLIDER_SPECS.daemonVoiceTranscriptQueueSize}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptQueueSize: n })
                }
              />
              <SliderRow
                label="Coalesce (ms)"
                value={config.daemonVoiceTranscriptCoalesceMs}
                spec={SLIDER_SPECS.daemonVoiceTranscriptCoalesceMs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptCoalesceMs: n })
                }
                formatDisplay={formatMs}
              />
              <SliderRow
                label="Max merge jobs"
                value={config.daemonVoiceTranscriptMaxMergeJobs}
                spec={SLIDER_SPECS.daemonVoiceTranscriptMaxMergeJobs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeJobs: n })
                }
              />
              <SliderRow
                label="Max merge chars"
                value={config.daemonVoiceTranscriptMaxMergeChars}
                spec={SLIDER_SPECS.daemonVoiceTranscriptMaxMergeChars}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeChars: n })
                }
              />
              <SliderRow
                label="Max agent turns"
                value={config.maxPlannerTurns}
                spec={SLIDER_SPECS.maxPlannerTurns}
                disabled={disabled}
                onCommit={(n) => patchConfig({ maxPlannerTurns: n })}
              />
              <SliderRow
                label="Max intent retries"
                value={config.maxIntentDispatchRetries}
                spec={SLIDER_SPECS.maxIntentDispatchRetries}
                disabled={disabled}
                onCommit={(n) => patchConfig({ maxIntentDispatchRetries: n })}
              />
              <SliderRow
                label="Max context rounds"
                value={config.maxContextRounds}
                spec={SLIDER_SPECS.maxContextRounds}
                disabled={disabled}
                onCommit={(n) => patchConfig({ maxContextRounds: n })}
              />
              <SliderRow
                label="Max context bytes"
                value={config.maxContextBytes}
                spec={SLIDER_SPECS.maxContextBytes}
                disabled={disabled}
                onCommit={(n) => patchConfig({ maxContextBytes: n })}
              />
              <SliderRow
                label="Max consecutive context requests"
                value={config.maxConsecutiveContextRequests}
                spec={SLIDER_SPECS.maxConsecutiveContextRequests}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({
                    maxConsecutiveContextRequests: n,
                  })
                }
              />
              <OptionalZeroSliderRow
                label="Session idle reset"
                hint="When off, voice sessions are not dropped after idle"
                toggleLabel="Drop voice session after idle timeout"
                value={config.sessionIdleResetMs}
                spec={SLIDER_SPECS.sessionIdleResetMs}
                disabled={disabled}
                enableDefault={1_800_000}
                onCommit={(n) => patchConfig({ sessionIdleResetMs: n })}
                formatDisplay={formatMs}
              />
            </div>
          </div>
        </AdvancedDisclosure>
      ) : null}

      {config ? (
        <section className="settings-section settings-section-after-advanced">
          <h2 className="settings-section-title">Debug</h2>
          <div className="settings-stack">
            <ToggleRow
              id="vocode-voice-vad-debug"
              label="Voice VAD debug"
              description="Forward VOCODE_VOICE_VAD_DEBUG to the sidecar (verbose stderr)."
              checked={config.voiceVadDebug}
              disabled={disabled}
              onChange={(voiceVadDebug) => patchConfig({ voiceVadDebug })}
            />
            <ToggleRow
              id="vocode-voice-protocol-log"
              label="Log sidecar protocol"
              description="Log JSON lines from the voice sidecar to Developer Tools."
              checked={config.voiceSidecarLogProtocol}
              disabled={disabled}
              onChange={(voiceSidecarLogProtocol) =>
                patchConfig({ voiceSidecarLogProtocol })
              }
            />
          </div>
        </section>
      ) : null}

      <button
        type="button"
        className="settings-open-vscode"
        disabled={!api}
        onClick={() => api?.postMessage({ type: "openExtensionSettings" })}
      >
        Open all Vocode settings in VS Code…
      </button>
    </div>
  );
}
