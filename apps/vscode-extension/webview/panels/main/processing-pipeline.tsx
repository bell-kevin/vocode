import type { PendingRow } from "../../types";

export type ProcessingStepVisual = "done" | "active" | "pending";

export function processingPipelineSteps(status: PendingRow["status"]): {
  label: string;
  visual: ProcessingStepVisual;
  title?: string;
}[] {
  const st = status;
  return [
    { label: "Transcript committed", visual: "done" },
    {
      label: "Understand & run",
      visual: st === "processing" ? "active" : "pending",
      title:
        st === "queued"
          ? "Waiting to process this line"
          : "Working on your request",
    },
  ];
}

export function ProcessingStepRow({
  label,
  visual,
  title,
}: {
  label: string;
  visual: ProcessingStepVisual;
  title?: string;
}) {
  return (
    <div className={`processing-step processing-step-${visual}`} title={title}>
      <span className="processing-step-mark" aria-hidden="true" />
      <span className="processing-step-label">{label}</span>
    </div>
  );
}
