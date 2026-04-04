import type { HandledRow } from "../../types";
import { fmtTime } from "../../util";

function SkippedCard({ h }: { h: HandledRow }) {
  const summary =
    typeof h.summary === "string" && h.summary.trim().length > 0
      ? h.summary.trim()
      : null;
  return (
    <div className="card skipped-card">
      <div className="meta">
        <span
          className="badge"
          title="Agent treated this line as not actionable"
        >
          Skipped
        </span>
        <span>{fmtTime(h.receivedAt)}</span>
      </div>
      {summary ? (
        <>
          <div className="text skipped-summary">{summary}</div>
          <div className="history-transcript muted-transcript">{h.text}</div>
        </>
      ) : (
        <div className="text muted-transcript">{h.text}</div>
      )}
    </div>
  );
}

export function SkippedSection({ items }: { items: readonly HandledRow[] }) {
  if (!items.length) {
    return null;
  }
  return (
    <section className="panel-section skipped-section">
      <details className="skipped-details">
        <summary>Skipped ({items.length})</summary>
        <div className="stack panel-section-body">
          {items.map((h) => (
            <SkippedCard key={`s-${h.receivedAt}-${h.text}`} h={h} />
          ))}
        </div>
      </details>
    </section>
  );
}
