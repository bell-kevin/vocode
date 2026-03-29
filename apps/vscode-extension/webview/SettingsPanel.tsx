import { useEffect, useState } from "react";

import type { VocodePanelConfig } from "./panel-config-types";
import { getVsCodeApi } from "./vscode-api";

export type PanelConfig = VocodePanelConfig;

function patchConfig(patch: Partial<VocodePanelConfig>) {
  getVsCodeApi()?.postMessage({ type: "setPanelConfig", patch });
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

function TextField(props: {
  label: string;
  hint?: string;
  value: string;
  disabled?: boolean;
  onCommit: (v: string) => void;
}) {
  const { label, hint, value, disabled, onCommit } = props;
  const [local, setLocal] = useState(value);
  useEffect(() => setLocal(value), [value]);
  return (
    <label className="settings-field">
      <span className="settings-field-label">{label}</span>
      {hint ? <span className="settings-field-hint">{hint}</span> : null}
      <input
        type="text"
        className="settings-input"
        disabled={disabled}
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        onBlur={() => onCommit(local)}
      />
    </label>
  );
}

function NumField(props: {
  label: string;
  hint?: string;
  value: number;
  step?: string;
  disabled?: boolean;
  onCommit: (n: number) => void;
}) {
  const { label, hint, value, step, disabled, onCommit } = props;
  const [local, setLocal] = useState(String(value));
  useEffect(() => setLocal(String(value)), [value]);
  return (
    <label className="settings-field">
      <span className="settings-field-label">{label}</span>
      {hint ? <span className="settings-field-hint">{hint}</span> : null}
      <input
        type="number"
        className="settings-input"
        step={step}
        disabled={disabled}
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        onBlur={() => {
          const n = Number(local);
          if (Number.isFinite(n)) {
            onCommit(n);
          }
        }}
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
      <p className="settings-intro">
        Values you set here override workspace <code>.env</code> for spawned
        daemon and voice processes. Restart voice (Stop → Start) and reload the
        window if the daemon is already running.
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
          <div className="settings-field-grid">
            <TextField
              label="STT language"
              hint='ISO 639-1 (e.g. en) or "auto"'
              value={config.elevenLabsSttLanguage}
              disabled={disabled}
              onCommit={(v) => patchConfig({ elevenLabsSttLanguage: v })}
            />
            <TextField
              label="STT model id"
              value={config.elevenLabsSttModelId}
              disabled={disabled}
              onCommit={(v) => patchConfig({ elevenLabsSttModelId: v })}
            />
            <NumField
              label="Commit response timeout (ms)"
              hint="0 = wait indefinitely for committed_transcript"
              value={config.voiceSttCommitResponseTimeoutMs}
              disabled={disabled}
              onCommit={(n) =>
                patchConfig({ voiceSttCommitResponseTimeoutMs: n })
              }
            />
          </div>
        </section>
      ) : null}

      {config ? (
        <section className="settings-section">
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

      {config ? (
        <details className="settings-advanced">
          <summary className="settings-advanced-summary">Advanced</summary>
          <div className="settings-advanced-body">
            <h3 className="settings-subhead">VAD</h3>
            <div className="settings-field-grid">
              <NumField
                label="Threshold multiplier"
                value={config.voiceVadThresholdMultiplier}
                step="0.05"
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ voiceVadThresholdMultiplier: n })
                }
              />
              <NumField
                label="Min energy floor"
                value={config.voiceVadMinEnergyFloor}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadMinEnergyFloor: n })}
              />
              <NumField
                label="Start (ms)"
                value={config.voiceVadStartMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadStartMs: n })}
              />
              <NumField
                label="End (ms)"
                value={config.voiceVadEndMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadEndMs: n })}
              />
              <NumField
                label="Preroll (ms)"
                value={config.voiceVadPrerollMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadPrerollMs: n })}
              />
            </div>
            <h3 className="settings-subhead">Stream</h3>
            <div className="settings-field-grid">
              <NumField
                label="Min chunk (ms)"
                value={config.voiceStreamMinChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMinChunkMs: n })}
              />
              <NumField
                label="Max chunk (ms)"
                value={config.voiceStreamMaxChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMaxChunkMs: n })}
              />
              <NumField
                label="Max utterance (ms)"
                hint="0 = off"
                value={config.voiceStreamMaxUtteranceMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMaxUtteranceMs: n })}
              />
            </div>
            <h3 className="settings-subhead">Daemon (voice transcript)</h3>
            <div className="settings-field-grid">
              <NumField
                label="Queue size"
                value={config.daemonVoiceTranscriptQueueSize}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptQueueSize: n })
                }
              />
              <NumField
                label="Coalesce (ms)"
                value={config.daemonVoiceTranscriptCoalesceMs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptCoalesceMs: n })
                }
              />
              <NumField
                label="Max merge jobs"
                value={config.daemonVoiceTranscriptMaxMergeJobs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeJobs: n })
                }
              />
              <NumField
                label="Max merge chars"
                value={config.daemonVoiceTranscriptMaxMergeChars}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeChars: n })
                }
              />
              <NumField
                label="Max agent turns"
                value={config.daemonVoiceMaxAgentTurns}
                disabled={disabled}
                onCommit={(n) => patchConfig({ daemonVoiceMaxAgentTurns: n })}
              />
              <NumField
                label="Max intent retries"
                value={config.daemonVoiceMaxIntentRetries}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceMaxIntentRetries: n })
                }
              />
              <NumField
                label="Max context rounds"
                value={config.daemonVoiceMaxContextRounds}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceMaxContextRounds: n })
                }
              />
              <NumField
                label="Max context bytes"
                value={config.daemonVoiceMaxContextBytes}
                disabled={disabled}
                onCommit={(n) => patchConfig({ daemonVoiceMaxContextBytes: n })}
              />
              <NumField
                label="Max consecutive context requests"
                value={config.daemonVoiceMaxConsecutiveContextRequests}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({
                    daemonVoiceMaxConsecutiveContextRequests: n,
                  })
                }
              />
              <NumField
                label="Session idle reset (ms)"
                hint="0 = off"
                value={config.daemonSessionIdleResetMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ daemonSessionIdleResetMs: n })}
              />
            </div>
          </div>
        </details>
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
