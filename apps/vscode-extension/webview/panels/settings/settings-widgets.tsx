import { type ReactNode, useEffect, useId, useState } from "react";

import type { VocodeConfig } from "../../config";

export const STT_LANGUAGE_PRESETS = [
  { iso: "en", name: "English" },
  { iso: "auto", name: "Auto" },
  { iso: "es", name: "Español" },
  { iso: "fr", name: "Français" },
  { iso: "de", name: "Deutsch" },
  { iso: "ja", name: "日本語" },
  { iso: "zh", name: "中文" },
  { iso: "pt", name: "Português" },
  { iso: "it", name: "Italiano" },
  { iso: "ko", name: "한국어" },
] as const;

export const DEFAULT_STT_MODEL_ID = "scribe_v2_realtime";

export type SliderSpec = {
  min: number;
  max: number;
  step: number;
};

export const SLIDER_SPECS: Record<
  keyof Pick<
    VocodeConfig,
    | "voiceVadThresholdMultiplier"
    | "voiceVadMinEnergyFloor"
    | "voiceVadStartMs"
    | "voiceVadEndMs"
    | "voiceVadPrerollMs"
    | "voiceStreamMinChunkMs"
    | "voiceStreamMaxChunkMs"
    | "voiceStreamMaxUtteranceMs"
    | "daemonVoiceTranscriptQueueSize"
    | "daemonVoiceTranscriptCoalesceMs"
    | "daemonVoiceTranscriptMaxMergeJobs"
    | "daemonVoiceTranscriptMaxMergeChars"
    | "sessionIdleResetMs"
    | "voiceSttCommitResponseTimeoutMs"
  >,
  SliderSpec
> = {
  voiceVadThresholdMultiplier: { min: 0.5, max: 3, step: 0.05 },
  voiceVadMinEnergyFloor: { min: 0, max: 500, step: 1 },
  voiceVadStartMs: { min: 20, max: 500, step: 5 },
  voiceVadEndMs: { min: 200, max: 3000, step: 25 },
  voiceVadPrerollMs: { min: 0, max: 1000, step: 10 },
  voiceStreamMinChunkMs: { min: 50, max: 2000, step: 10 },
  voiceStreamMaxChunkMs: { min: 100, max: 5000, step: 25 },
  voiceStreamMaxUtteranceMs: { min: 500, max: 120_000, step: 500 },
  daemonVoiceTranscriptQueueSize: { min: 1, max: 50, step: 1 },
  daemonVoiceTranscriptCoalesceMs: { min: 0, max: 5000, step: 25 },
  daemonVoiceTranscriptMaxMergeJobs: { min: 1, max: 20, step: 1 },
  daemonVoiceTranscriptMaxMergeChars: { min: 500, max: 50_000, step: 100 },
  sessionIdleResetMs: { min: 60_000, max: 14_400_000, step: 60_000 },
  voiceSttCommitResponseTimeoutMs: { min: 1000, max: 180_000, step: 500 },
};

function clamp(n: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, n));
}

function formatSliderValue(n: number, step: number): string {
  if (step >= 1 && Number.isInteger(step)) {
    return String(Math.round(n));
  }
  return n.toFixed(2).replace(/\.?0+$/, "");
}

export function SliderRow(props: {
  label: string;
  hint?: string;
  value: number;
  spec: SliderSpec;
  disabled?: boolean;
  onCommit: (n: number) => void;
  /** Large values shown compactly (e.g. ms → minutes) */
  formatDisplay?: (n: number) => string;
  /** Only the range input (for parents that render their own title/value header). */
  variant?: "default" | "trackOnly";
  /** Used with `trackOnly` so an outer label can point at the control. */
  id?: string;
}) {
  const {
    label,
    hint,
    value,
    spec,
    disabled,
    onCommit,
    formatDisplay,
    variant = "default",
    id: idProp,
  } = props;
  const genId = useId();
  const id = idProp ?? genId;
  const { min, max, step } = spec;
  const safe = clamp(Number.isFinite(value) ? value : min, min, max);

  const rangeInput = (
    <input
      id={id}
      type="range"
      className="settings-range"
      min={min}
      max={max}
      step={step}
      disabled={disabled}
      value={safe}
      onChange={(e) => {
        const n = Number(e.target.value);
        if (Number.isFinite(n)) {
          onCommit(n);
        }
      }}
    />
  );

  if (variant === "trackOnly") {
    return (
      <div className="settings-field settings-field-slider settings-field-slider-nested">
        {rangeInput}
      </div>
    );
  }

  return (
    <div
      className={`settings-field settings-field-slider${label ? "" : " settings-field-slider-nested"}`}
    >
      <div className="settings-slider-header">
        {label ? (
          <label className="settings-field-label" htmlFor={id}>
            {label}
          </label>
        ) : (
          <span className="settings-field-label settings-sr-only">Value</span>
        )}
        <span className="settings-slider-value" aria-live="polite">
          {formatDisplay ? formatDisplay(safe) : formatSliderValue(safe, step)}
        </span>
      </div>
      {hint ? <span className="settings-field-hint">{hint}</span> : null}
      {rangeInput}
    </div>
  );
}

interface OptionalZeroSliderRowProps {
  label: string;
  hint?: string;
  toggleLabel: string;
  value: number;
  spec: SliderSpec;
  disabled?: boolean;
  onCommit: (n: number) => void;
  /** Value applied when the user turns the feature on from 0 */
  enableDefault: number;
  formatDisplay?: (n: number) => string;
  /** Shown on the right when value is 0 (e.g. "Unlimited" vs "Off"). */
  offSummary?: string;
}
export function OptionalZeroSliderRow({
  label,
  hint,
  toggleLabel,
  value,
  spec,
  disabled,
  onCommit,
  enableDefault,
  formatDisplay,
  offSummary = "Off",
}: OptionalZeroSliderRowProps) {
  const enabled = value !== 0;
  const rangeId = useId();
  const safe = clamp(
    Number.isFinite(value) ? value : spec.min,
    spec.min,
    spec.max,
  );
  const valueSummary = enabled
    ? formatDisplay
      ? formatDisplay(safe)
      : formatSliderValue(safe, spec.step)
    : offSummary;

  return (
    <div className="settings-field settings-optional-slider-block">
      <div className="settings-slider-header">
        <label
          className="settings-field-label"
          htmlFor={enabled ? rangeId : undefined}
        >
          {label}
        </label>
        <span className="settings-slider-value" aria-live="polite">
          {valueSummary}
        </span>
      </div>
      {hint ? <span className="settings-field-hint">{hint}</span> : null}
      <label className="settings-inline-toggle">
        <input
          type="checkbox"
          className="settings-toggle"
          checked={enabled}
          disabled={disabled}
          onChange={(e) => {
            if (e.target.checked) {
              const d = enableDefault > 0 ? enableDefault : spec.min;
              onCommit(clamp(d, spec.min, spec.max));
            } else {
              onCommit(0);
            }
          }}
        />
        <span>{toggleLabel}</span>
      </label>
      {enabled ? (
        <SliderRow
          label=""
          variant="trackOnly"
          id={rangeId}
          value={value}
          spec={spec}
          disabled={disabled}
          onCommit={onCommit}
          formatDisplay={formatDisplay}
        />
      ) : null}
    </div>
  );
}

function isSttLanguagePreset(value: string): boolean {
  return STT_LANGUAGE_PRESETS.some((lang) => lang.iso === value);
}

export function LanguageSelectRow(props: {
  value: string;
  disabled?: boolean;
  onCommit: (v: string) => void;
}) {
  const { value, disabled, onCommit } = props;
  const isPreset = isSttLanguagePreset(value);
  const [customOpen, setCustomOpen] = useState(!isPreset);
  const [customDraft, setCustomDraft] = useState(value);
  const selectId = useId();
  const customId = useId();

  useEffect(() => {
    if (isSttLanguagePreset(value)) {
      setCustomOpen(false);
    } else {
      setCustomOpen(true);
      setCustomDraft(value);
    }
  }, [value]);

  return (
    <div className="settings-field">
      <label className="settings-field-label" htmlFor={selectId}>
        STT language
      </label>
      <select
        id={selectId}
        className="settings-select"
        disabled={disabled}
        value={customOpen ? "__custom__" : value}
        onChange={(e) => {
          const v = e.target.value;
          if (v === "__custom__") {
            setCustomOpen(true);
            setCustomDraft(value);
            return;
          }
          setCustomOpen(false);
          onCommit(v);
        }}
      >
        {STT_LANGUAGE_PRESETS.map((code) => (
          <option key={code.iso} value={code.iso}>
            {code.name}
          </option>
        ))}
        <option value="__custom__">Other…</option>
      </select>
      {customOpen ? (
        <>
          <span className="settings-field-hint">
            Enter ElevenLabs language_code (ISO 639-1)
          </span>
          <input
            id={customId}
            type="text"
            className="settings-input settings-input-mt"
            disabled={disabled}
            value={customDraft}
            placeholder="e.g. nl"
            onChange={(e) => setCustomDraft(e.target.value)}
            onBlur={() => {
              const t = customDraft.trim();
              if (t) {
                onCommit(t);
              }
            }}
          />
        </>
      ) : null}
    </div>
  );
}

export function SttModelChoiceRow(props: {
  value: string;
  disabled?: boolean;
  onCommit: (v: string) => void;
}) {
  const { value, disabled, onCommit } = props;
  const isDefault = value === DEFAULT_STT_MODEL_ID;
  const [customArmed, setCustomArmed] = useState(!isDefault);
  const [draft, setDraft] = useState(isDefault ? "" : value);

  useEffect(() => {
    if (value === DEFAULT_STT_MODEL_ID) {
      setCustomArmed(false);
      setDraft("");
    } else {
      setCustomArmed(true);
      setDraft(value);
    }
  }, [value]);

  return (
    <div className="settings-field">
      <span className="settings-field-label">STT model</span>
      <span className="settings-field-hint">
        Realtime WebSocket model id (see ElevenLabs docs)
      </span>
      <div className="settings-segmented" role="group" aria-label="STT model">
        <button
          type="button"
          className={`settings-segment ${isDefault && !customArmed ? "settings-segment-active" : ""}`}
          disabled={disabled}
          onClick={() => {
            setCustomArmed(false);
            onCommit(DEFAULT_STT_MODEL_ID);
          }}
        >
          Scribe v2 (realtime)
        </button>
        <button
          type="button"
          className={`settings-segment ${!isDefault || customArmed ? "settings-segment-active" : ""}`}
          disabled={disabled}
          onClick={() => setCustomArmed(true)}
        >
          Custom ID
        </button>
      </div>
      {!isDefault || customArmed ? (
        <input
          type="text"
          className="settings-input settings-input-mt"
          disabled={disabled}
          value={isDefault ? draft : value}
          placeholder="model_id"
          onChange={(e) => {
            const v = e.target.value;
            if (isDefault) {
              setDraft(v);
            } else {
              onCommit(v);
            }
          }}
          onBlur={() => {
            if (!isDefault) {
              return;
            }
            const t = draft.trim();
            if (t) {
              onCommit(t);
            } else {
              setCustomArmed(false);
            }
          }}
        />
      ) : null}
    </div>
  );
}

export function AdvancedDisclosure(props: {
  title: string;
  children: ReactNode;
  defaultOpen?: boolean;
}) {
  const { title, children, defaultOpen = false } = props;
  const [open, setOpen] = useState(defaultOpen);
  const panelId = useId();

  return (
    <div className="settings-disclosure">
      <button
        type="button"
        className="settings-disclosure-trigger"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((o) => !o)}
      >
        <span
          className="settings-disclosure-chevron"
          aria-hidden
          data-open={open}
        />
        <span className="settings-disclosure-title">{title}</span>
      </button>
      {open ? (
        <div
          id={panelId}
          className="settings-disclosure-panel"
          role="region"
          aria-label={title}
        >
          {children}
        </div>
      ) : null}
    </div>
  );
}
