import type { HandledRow } from "../../types";
import { fmtTime } from "../../util";

function HistoryCard({ h }: { h: HandledRow }) {
  const failed =
    typeof h.errorMessage === "string" && h.errorMessage.length > 0;
  const summary =
    typeof h.summary === "string" && h.summary.trim().length > 0
      ? h.summary.trim()
      : null;

  if (failed) {
    return (
      <div className="card done failed history-card">
        <div className="meta">
          <span className="badge" title="Processing did not succeed">
            {"Couldn't run"}
          </span>
          <span>{fmtTime(h.receivedAt)}</span>
        </div>
        {summary ? <div className="text history-summary">{summary}</div> : null}
        <div className="history-transcript muted-transcript">{h.text}</div>
        <div className="error-detail">Error: {h.errorMessage}</div>
      </div>
    );
  }

  return (
    <div className="card done history-card">
      <div className="meta">
        <span>{fmtTime(h.receivedAt)}</span>
      </div>
      {summary ? (
        <>
          <div className="text history-summary">{summary}</div>
          <div className="history-transcript muted-transcript">{h.text}</div>
        </>
      ) : (
        <div className="text">{h.text}</div>
      )}
    </div>
  );
}

export function HistorySection({ items }: { items: readonly HandledRow[] }) {
  return (
    <section className="panel-section">
      <h1>Recent</h1>
      {items.length > 0 ? (
        <div className="stack">
          {items.map((h) => (
            <HistoryCard key={`h-${h.receivedAt}-${h.text}`} h={h} />
          ))}
        </div>
      ) : null}
    </section>
  );
}
