import { AudioInputMeter } from "./audio-meter";
import type { PanelState } from "./types";

type Props = {
  state: PanelState;
  className?: string;
};

/**
 * Unified voice UI: input level, waveform, and streaming partial transcript in one surface.
 */
export function VoiceVisualization(props: Props) {
  const { state, className } = props;
  const voiceListening = state.voiceListening === true;
  const speaking = state.audioMeter.speaking === true;
  const partialRaw =
    typeof state.latestPartial === "string" ? state.latestPartial : "";
  const partial = partialRaw.trim().length > 0 ? partialRaw : null;

  const showDraftChrome = voiceListening && partial !== null;

  const statusMain = !voiceListening
    ? "Idle"
    : speaking
      ? "Speaking"
      : "Listening";
  const statusSub = !voiceListening
    ? "Start voice to capture speech"
    : speaking
      ? "Audio detected"
      : "Waiting for speech";

  const wrapClass = ["voice-visualization", className]
    .filter(Boolean)
    .join(" ");

  return (
    <section className={wrapClass} aria-label="Voice input and live transcript">
      <div className="voice-viz-shell">
        <header className="voice-viz-header">
          <div className="voice-viz-header-main">
            <span className="voice-viz-eyebrow">Voice</span>
            <div className="voice-viz-status-row">
              <span
                className={[
                  "voice-viz-orb",
                  voiceListening && speaking ? "voice-viz-orb-active" : "",
                  voiceListening && !speaking ? "voice-viz-orb-listen" : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
                aria-hidden
              />
              <div className="voice-viz-status-text">
                <span className="badge voice-viz-badge">{statusMain}</span>
                <span className="voice-viz-sub">{statusSub}</span>
              </div>
            </div>
          </div>
        </header>

        <AudioInputMeter state={state} className="voice-viz-meter" />

        {showDraftChrome ? (
          <div className="voice-viz-draft">
            <div className="voice-viz-draft-label">
              <span>Live draft</span>
              <span
                className="voice-viz-draft-hint"
                title="Preview only—text is finalized after you pause or finish the phrase"
              >
                Live transcription
              </span>
            </div>
            {partial !== null ? (
              <p className="voice-viz-draft-text" aria-live="polite">
                {partial}
              </p>
            ) : null}
            {partial !== null && (
              <div className="typing voice-viz-typing" aria-hidden="true">
                <span className="dot" />
                <span className="dot" />
                <span className="dot" />
              </div>
            )}
          </div>
        ) : null}
      </div>
    </section>
  );
}
