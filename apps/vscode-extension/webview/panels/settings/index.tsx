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
      "Speech-to-text is not set up yet. Add an ElevenLabs API key under API keys to use voice.",
    );
  }
  if (!config.daemonAgentProvider) {
    items.push(
      "No AI provider is selected. Choose one under AI for voice commands, or use Built-in to run without a cloud model.",
    );
  }
  if (config.daemonAgentProvider === "openai") {
    if (!config.openaiApiKeyConfigured) {
      items.push(
        "OpenAI is selected, but no OpenAI API key is saved. Add one under API keys so commands use OpenAI; otherwise Vocode uses the built-in agent.",
      );
    }
    if (!config.daemonOpenaiModel) {
      items.push(
        "OpenAI is selected, but no model is chosen. Pick a model under AI for voice commands.",
      );
    }
  }
  if (config.daemonAgentProvider === "anthropic") {
    if (!config.anthropicApiKeyConfigured) {
      items.push(
        "Anthropic is selected, but no Anthropic API key is saved. Add one under API keys so commands use Anthropic; otherwise Vocode uses the built-in agent.",
      );
    }
    if (!config.daemonAnthropicModel) {
      items.push(
        "Anthropic is selected, but no model is chosen. Pick a model under AI for voice commands.",
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
            ElevenLabs (speech-to-text)
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
        Adjust listening, speech recognition, and how voice commands are run.
        Changes apply as you go.
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
          <h2 className="settings-section-title">AI for voice commands</h2>
          <p className="settings-subtle">
            Chooses which model interprets what you say and plans edits.
            Built-in works offline without an API key; OpenAI and Anthropic use
            the keys below.
          </p>
          <div className="settings-field-stack">
            <div className="settings-field">
              <span className="settings-field-label">Provider</span>
              <div
                className="settings-segmented"
                role="group"
                aria-label="AI provider for voice commands"
              >
                <button
                  type="button"
                  className={`settings-segment ${config.daemonAgentProvider === "stub" ? "settings-segment-active" : ""}`}
                  disabled={disabled}
                  onClick={() => patchConfig({ daemonAgentProvider: "stub" })}
                >
                  Built-in
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
          <h2 className="settings-section-title">Speech recognition</h2>
          <p className="settings-subtle">
            Powered by ElevenLabs. To bias recognition toward names or jargon,
            add a <code>.vocode</code> file at a folder root with a{" "}
            <code>sttKeywords</code> list. Use the Command Palette:{" "}
            <strong>Vocode: Create Workspace .vocode File</strong>, then save
            and restart if prompted.
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
              label="Final transcript wait time"
              hint="After you pause speaking, how long to wait for the final text before sending more audio. Turn off to wait as long as the service needs."
              toggleLabel="Limit wait for final transcript after each phrase"
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
        <AdvancedDisclosure title="Advanced tuning">
          <div className="settings-advanced-inner">
            <h3 className="settings-subhead">When Vocode hears speech</h3>
            <p className="settings-subtle">
              Fine-tune how your microphone is interpreted. Only change these if
              something is cutting off too early, picking up noise, or feeling
              sluggish.
            </p>
            <div className="settings-field-stack">
              <SliderRow
                label="Speech vs noise sensitivity"
                hint="Higher = louder speech required. Use in noisy rooms; lower if your voice is quiet."
                value={config.voiceVadThresholdMultiplier}
                spec={SLIDER_SPECS.voiceVadThresholdMultiplier}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ voiceVadThresholdMultiplier: n })
                }
              />
              <SliderRow
                label="Ignore very quiet sound"
                hint="Raises the floor so hum and background noise are less likely to count as speech."
                value={config.voiceVadMinEnergyFloor}
                spec={SLIDER_SPECS.voiceVadMinEnergyFloor}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadMinEnergyFloor: n })}
              />
              <SliderRow
                label="Time to confirm you started talking"
                hint="How long speech must continue before a phrase officially starts. Higher reduces accidental triggers."
                value={config.voiceVadStartMs}
                spec={SLIDER_SPECS.voiceVadStartMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadStartMs: n })}
                formatDisplay={formatMs}
              />
              <SliderRow
                label="Pause before end of phrase"
                hint="How long you must pause before Vocode treats the sentence as finished."
                value={config.voiceVadEndMs}
                spec={SLIDER_SPECS.voiceVadEndMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadEndMs: n })}
                formatDisplay={formatMs}
              />
              <SliderRow
                label="Audio kept before speech starts"
                hint="Includes a little sound before the detected start so words are not clipped."
                value={config.voiceVadPrerollMs}
                spec={SLIDER_SPECS.voiceVadPrerollMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceVadPrerollMs: n })}
                formatDisplay={formatMs}
              />
            </div>
            <h3 className="settings-subhead">Audio sent for recognition</h3>
            <p className="settings-subtle">
              Controls how audio is sliced before it is sent for transcription.
              Defaults work for most setups.
            </p>
            <div className="settings-field-stack">
              <SliderRow
                label="Smallest slice (ms)"
                hint="Minimum amount of audio per send. Affects how often partial results can update."
                value={config.voiceStreamMinChunkMs}
                spec={SLIDER_SPECS.voiceStreamMinChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMinChunkMs: n })}
                formatDisplay={formatMs}
              />
              <SliderRow
                label="Largest slice (ms)"
                hint="Maximum time window bundled into one send before it must go out."
                value={config.voiceStreamMaxChunkMs}
                spec={SLIDER_SPECS.voiceStreamMaxChunkMs}
                disabled={disabled}
                onCommit={(n) => patchConfig({ voiceStreamMaxChunkMs: n })}
                formatDisplay={formatMs}
              />
              <OptionalZeroSliderRow
                label="Break up very long talking (ms)"
                hint="When on, Vocode finalizes text periodically during one long monologue so nothing is held forever. When off, it waits for a pause or when you stop voice."
                toggleLabel="Periodically finalize text during long speech"
                value={config.voiceStreamMaxUtteranceMs}
                spec={SLIDER_SPECS.voiceStreamMaxUtteranceMs}
                disabled={disabled}
                enableDefault={8000}
                onCommit={(n) => patchConfig({ voiceStreamMaxUtteranceMs: n })}
                formatDisplay={formatMs}
                offSummary="Only on pause or stop"
              />
            </div>
            <h3 className="settings-subhead">Batching voice commands</h3>
            <p className="settings-subtle">
              How Vocode combines back-to-back phrases and how many can wait in
              line. Change only if you need different throughput or merging
              behavior.
            </p>
            <div className="settings-field-stack">
              <SliderRow
                label="Command queue size"
                hint="How many spoken commands can wait to run at once."
                value={config.daemonVoiceTranscriptQueueSize}
                spec={SLIDER_SPECS.daemonVoiceTranscriptQueueSize}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptQueueSize: n })
                }
              />
              <SliderRow
                label="Combine phrases within (ms)"
                hint="Speech arriving in this window may be merged into one command."
                value={config.daemonVoiceTranscriptCoalesceMs}
                spec={SLIDER_SPECS.daemonVoiceTranscriptCoalesceMs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptCoalesceMs: n })
                }
                formatDisplay={formatMs}
              />
              <SliderRow
                label="Max lines merged at once"
                hint="Upper limit on how many separate spoken lines join one batch."
                value={config.daemonVoiceTranscriptMaxMergeJobs}
                spec={SLIDER_SPECS.daemonVoiceTranscriptMaxMergeJobs}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeJobs: n })
                }
              />
              <SliderRow
                label="Max characters merged at once"
                hint="Caps total text size when merging lines into one batch."
                value={config.daemonVoiceTranscriptMaxMergeChars}
                spec={SLIDER_SPECS.daemonVoiceTranscriptMaxMergeChars}
                disabled={disabled}
                onCommit={(n) =>
                  patchConfig({ daemonVoiceTranscriptMaxMergeChars: n })
                }
              />
              <OptionalZeroSliderRow
                label="Forget in-progress voice session after idle"
                hint="If you pause voice commands for this long, Vocode clears what it was holding for your current conversation—like a follow-up question, a file-picker step, or other remembered context. Your microphone and speech-to-text are not affected."
                toggleLabel="Clear remembered voice-session context after idle"
                value={config.sessionIdleResetMs}
                spec={SLIDER_SPECS.sessionIdleResetMs}
                disabled={disabled}
                enableDefault={1_800_000}
                onCommit={(n) => patchConfig({ sessionIdleResetMs: n })}
                formatDisplay={formatMs}
                offSummary="No idle timeout"
              />
            </div>
          </div>
        </AdvancedDisclosure>
      ) : null}

      {config ? (
        <section className="settings-section settings-section-after-advanced">
          <h2 className="settings-section-title">Troubleshooting</h2>
          <p className="settings-subtle">
            Extra logging for diagnosing voice issues. Leave off during normal
            use.
          </p>
          <div className="settings-stack">
            <ToggleRow
              id="vocode-voice-vad-debug"
              label="Verbose speech-detection log"
              description="Print detailed mic / speech-boundary logs from the voice helper (helpful when tuning or reporting bugs)."
              checked={config.voiceVadDebug}
              disabled={disabled}
              onChange={(voiceVadDebug) => patchConfig({ voiceVadDebug })}
            />
            <ToggleRow
              id="vocode-voice-protocol-log"
              label="Log raw voice protocol"
              description="Log every message between VS Code and the voice helper in Developer Tools. Very noisy."
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
