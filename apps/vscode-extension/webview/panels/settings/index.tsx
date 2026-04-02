import { useState } from "react";

import { getVsCodeApi } from "../../api/vscode";
import type { VocodeConfig } from "../../config";
import {
  AdvancedDisclosure,
  LanguageSelectRow,
  OptionalZeroSliderRow,
  SLIDER_SPECS,
  SliderRow,
  SttModelChoiceRow,
} from "./settings-widgets";

function patchConfig(patch: Partial<VocodeConfig>) {
  getVsCodeApi()?.postMessage({ type: "setPanelConfig", patch });
}

function formatMs(ms: number): string {
  if (ms >= 60_000) {
    const m = ms / 60_000;
    return Number.isInteger(m) ? `${m} min` : `${m.toFixed(1)} min`;
  }
  return `${ms} ms`;
}

function LlmWarnings({ config }: { config: VocodeConfig | null }) {
  if (!config) {
    return null;
  }
  const items: string[] = [];
  if (!config.elevenLabsApiKeyConfigured) {
    items.push(
      "ElevenLabs STT is not configured yet. Add an API key under API Keys to enable voice.",
    );
  }
  if (!config.daemonAgentProvider) {
    items.push(
      "No LLM provider is selected. Choose one under LLM Agent or leave it on Stub to run without a cloud model.",
    );
  }
  if (config.daemonAgentProvider === "openai") {
    if (!config.openaiApiKeyConfigured) {
      items.push(
        "OpenAI provider is selected, but no OpenAI API key is configured. Add one under API Keys for the agent to use OpenAI; otherwise it will fall back to a stub model.",
      );
    }
    if (!config.daemonOpenaiModel) {
      items.push(
        "OpenAI is selected, but no model is set. Pick a model under LLM Agent so the agent knows which OpenAI model to call.",
      );
    }
  }
  if (config.daemonAgentProvider === "anthropic") {
    if (!config.anthropicApiKeyConfigured) {
      items.push(
        "Anthropic provider is selected, but no Anthropic API key is configured. Add one under API Keys for the agent to use Anthropic; otherwise it will fall back to a stub model.",
      );
    }
    if (!config.daemonAnthropicModel) {
      items.push(
        "Anthropic is selected, but no model is set. Pick a model under LLM Agent so the agent knows which Anthropic model to call.",
      );
    }
  }
  if (items.length === 0) {
    return null;
  }
  return (
    <>
      {items.map((text) => (
        <p key={text} className="settings-subtle">
          {text}
        </p>
      ))}
    </>
  );
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

function ApiKeysDisclosure(props: {
  config: VocodeConfig;
  disabled: boolean;
  keyDraft: string;
  setKeyDraft: (v: string) => void;
  openaiDraft: string;
  setOpenaiDraft: (v: string) => void;
  anthropicDraft: string;
  setAnthropicDraft: (v: string) => void;
}) {
  const {
    config,
    disabled,
    keyDraft,
    setKeyDraft,
    openaiDraft,
    setOpenaiDraft,
    anthropicDraft,
    setAnthropicDraft,
  } = props;
  const api = getVsCodeApi();

  return (
    <AdvancedDisclosure title="API Keys">
      <div className="settings-advanced-inner">
        <div className="settings-field">
          <label
            className="settings-field-label"
            htmlFor="vocode-elevenlabs-key"
          >
            ElevenLabs API key (voice)
          </label>
          <div className="settings-api-row">
            <input
              id="vocode-elevenlabs-key"
              type="password"
              className="settings-input settings-input-grow"
              autoComplete="off"
              placeholder={
                config.elevenLabsApiKeyConfigured
                  ? "••••••••"
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
        </div>

        <div className="settings-field">
          <label className="settings-field-label" htmlFor="vocode-openai-key">
            OpenAI API key
          </label>
          <div className="settings-api-row">
            <input
              id="vocode-openai-key"
              type="password"
              className="settings-input settings-input-grow"
              autoComplete="off"
              placeholder={
                config.openaiApiKeyConfigured ? "••••••••" : "OpenAI API key"
              }
              disabled={disabled}
              value={openaiDraft}
              onChange={(e) => setOpenaiDraft(e.target.value)}
            />
            <button
              type="button"
              className="settings-btn"
              disabled={disabled || !openaiDraft.trim()}
              onClick={() => {
                api?.postMessage({
                  type: "setOpenAIApiKey",
                  value: openaiDraft.trim(),
                });
                setOpenaiDraft("");
              }}
            >
              Save key
            </button>
            <button
              type="button"
              className="settings-btn settings-btn-ghost"
              disabled={disabled || !config.openaiApiKeyConfigured}
              onClick={() =>
                api?.postMessage({ type: "setOpenAIApiKey", value: "" })
              }
            >
              Remove
            </button>
          </div>
        </div>

        <div className="settings-field">
          <label
            className="settings-field-label"
            htmlFor="vocode-anthropic-key"
          >
            Anthropic API key
          </label>
          <div className="settings-api-row">
            <input
              id="vocode-anthropic-key"
              type="password"
              className="settings-input settings-input-grow"
              autoComplete="off"
              placeholder={
                config.anthropicApiKeyConfigured
                  ? "••••••••"
                  : "Anthropic API key"
              }
              disabled={disabled}
              value={anthropicDraft}
              onChange={(e) => setAnthropicDraft(e.target.value)}
            />
            <button
              type="button"
              className="settings-btn"
              disabled={disabled || !anthropicDraft.trim()}
              onClick={() => {
                api?.postMessage({
                  type: "setAnthropicApiKey",
                  value: anthropicDraft.trim(),
                });
                setAnthropicDraft("");
              }}
            >
              Save key
            </button>
            <button
              type="button"
              className="settings-btn settings-btn-ghost"
              disabled={disabled || !config.anthropicApiKeyConfigured}
              onClick={() =>
                api?.postMessage({ type: "setAnthropicApiKey", value: "" })
              }
            >
              Remove
            </button>
          </div>
        </div>
      </div>
    </AdvancedDisclosure>
  );
}

export function SettingsPanel(props: { config: VocodeConfig | null }) {
  const { config } = props;
  const api = getVsCodeApi();
  const disabled = !api || config === null;
  const [keyDraft, setKeyDraft] = useState("");
  const [openaiDraft, setOpenaiDraft] = useState("");
  const [anthropicDraft, setAnthropicDraft] = useState("");

  const openaiModels = ["gpt-4o-mini", "gpt-4o", "gpt-4.1-mini", "gpt-4.1"];
  const anthropicModels = [
    "claude-3-5-haiku-latest",
    "claude-3-5-sonnet-latest",
  ];

  return (
    <div className="settings-root">
      {config === null ? (
        <p className="settings-loading">Loading options…</p>
      ) : null}
      {config && !config.elevenLabsApiKeyConfigured ? (
        <div className="settings-banner" role="status">
          Add your ElevenLabs API key below to use voice.
        </div>
      ) : null}

      <p className="settings-intro-short">
        Vocode configuration. Changes apply automatically.
      </p>
      <LlmWarnings config={config} />

      {config ? (
        <ApiKeysDisclosure
          config={config}
          disabled={disabled}
          keyDraft={keyDraft}
          setKeyDraft={setKeyDraft}
          openaiDraft={openaiDraft}
          setOpenaiDraft={setOpenaiDraft}
          anthropicDraft={anthropicDraft}
          setAnthropicDraft={setAnthropicDraft}
        />
      ) : null}

      {config ? (
        <section className="settings-section">
          <h2 className="settings-section-title">LLM Agent</h2>
          <div className="settings-field-stack">
            <div className="settings-field">
              <span className="settings-field-label">Provider</span>
              <div
                className="settings-segmented"
                role="group"
                aria-label="LLM provider"
              >
                <button
                  type="button"
                  className={`settings-segment ${config.daemonAgentProvider === "stub" ? "settings-segment-active" : ""}`}
                  disabled={disabled}
                  onClick={() => patchConfig({ daemonAgentProvider: "stub" })}
                >
                  Stub
                </button>
                <button
                  type="button"
                  className={`settings-segment ${config.daemonAgentProvider === "openai" ? "settings-segment-active" : ""}`}
                  disabled={disabled}
                  onClick={() => patchConfig({ daemonAgentProvider: "openai" })}
                >
                  OpenAI
                </button>
                <button
                  type="button"
                  className={`settings-segment ${config.daemonAgentProvider === "anthropic" ? "settings-segment-active" : ""}`}
                  disabled={disabled}
                  onClick={() =>
                    patchConfig({ daemonAgentProvider: "anthropic" })
                  }
                >
                  Anthropic
                </button>
              </div>
            </div>

            {config.daemonAgentProvider === "openai" ? (
              <div className="settings-field">
                <label
                  className="settings-field-label"
                  htmlFor="vocode-openai-model"
                >
                  Model
                </label>
                <select
                  id="vocode-openai-model"
                  className="settings-select"
                  disabled={disabled}
                  value={config.daemonOpenaiModel}
                  onChange={(e) =>
                    patchConfig({ daemonOpenaiModel: e.target.value })
                  }
                >
                  {openaiModels.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </select>
              </div>
            ) : config.daemonAgentProvider === "anthropic" ? (
              <div className="settings-field">
                <label
                  className="settings-field-label"
                  htmlFor="vocode-anthropic-model"
                >
                  Model
                </label>
                <select
                  id="vocode-anthropic-model"
                  className="settings-select"
                  disabled={disabled}
                  value={config.daemonAnthropicModel}
                  onChange={(e) =>
                    patchConfig({ daemonAnthropicModel: e.target.value })
                  }
                >
                  {anthropicModels.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}
          </div>
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
