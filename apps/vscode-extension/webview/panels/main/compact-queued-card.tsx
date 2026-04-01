import type { PendingRow } from "../../types";
import { fmtTime, statusBadgeTitle, statusLabel } from "../../util";

export function CompactQueuedCard({ p }: { p: PendingRow }) {
  return (
    <div
      className={`card pending-compact pending processing-queued-card ${p.status}`}
    >
      <div className="meta">
        <span className="badge" title={statusBadgeTitle(p.status) || undefined}>
          {statusLabel(p.status)}
        </span>
        <span>{fmtTime(p.receivedAt)}</span>
      </div>
      <div className="text pending-compact-text">{p.text}</div>
    </div>
  );
}
