import type { PanelState } from "../../types";
import { ApplyingSection } from "./applying-section";
import { ChatSection } from "./chat-section";
import { HistorySection } from "./history-section";
import { SkippedSection } from "./skipped-section";

export function MainPanel({ state }: { state: PanelState }) {
  const pending = Array.isArray(state.pending) ? state.pending : [];
  const recentHandled = Array.isArray(state.recentHandled)
    ? state.recentHandled
    : [];
  const skippedItems = recentHandled.filter((h) => h.skipped === true);
  const historyItems = recentHandled.filter(
    (h) => h.skipped !== true && h.transcriptOutcome !== "answer",
  );

  return (
    <div id="main-root">
      <ChatSection state={state} />
      <ApplyingSection pending={pending} />
      <HistorySection items={historyItems} />
      <SkippedSection items={skippedItems} />
      <p className="hint">Vocode · Speak code</p>
    </div>
  );
}
