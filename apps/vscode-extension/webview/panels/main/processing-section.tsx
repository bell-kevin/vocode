import type { PendingRow } from "../../types";
import { fmtTime, statusBadgeTitle, statusLabel } from "../../util";
import { CompactQueuedCard } from "./compact-queued-card";
import {
  ProcessingStepRow,
  processingPipelineSteps,
} from "./processing-pipeline";

export function ProcessingSection({
  pending,
}: {
  pending: readonly PendingRow[];
}) {
  const primary = pending[0];
  const queuedRest = pending.length > 1 ? pending.slice(1) : [];

  return (
    <section className="panel-section">
      <h1>Processing</h1>
      {primary ? (
        <div className="stack processing-stack">
          <div
            className={`card pending processing-primary-card ${primary.status}`}
          >
            <div className="meta">
              <span
                className="badge"
                title={statusBadgeTitle(primary.status) || undefined}
              >
                {statusLabel(primary.status)}
              </span>
              <span>{fmtTime(primary.receivedAt)}</span>
            </div>
            <div className="text">{primary.text}</div>
            <div className="processing-steps" role="list" aria-label="Pipeline">
              {processingPipelineSteps(primary.status).map((s) => (
                <ProcessingStepRow key={s.label} {...s} />
              ))}
            </div>
          </div>
          {queuedRest.length > 0 ? (
            <div className="processing-queue-block">
              <h2 className="processing-subhead">
                Queued ({queuedRest.length})
              </h2>
              <div className="stack processing-queue-stack">
                {queuedRest.map((p) => (
                  <CompactQueuedCard key={p.id} p={p} />
                ))}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
