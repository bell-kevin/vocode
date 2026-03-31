import type { DirectiveApplyChecklistRowState, PanelState } from "./types";

export function fmtTime(iso: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }).format(new Date(iso));
  } catch {
    return "";
  }
}

export function statusLabel(status: string): string {
  switch (status) {
    case "queued":
      return "Queued";
    case "processing":
      return "Applying…";
    default:
      return status;
  }
}

export function statusBadgeTitle(status: string): string {
  switch (status) {
    case "queued":
      return "Committed transcript — waiting to run in the workspace";
    case "processing":
      return "Running this committed line through the Vocode agent";
    default:
      return "";
  }
}

export function normalizePanelState(raw: unknown): PanelState {
  if (!raw || typeof raw !== "object") {
    return emptyState();
  }
  const o = raw as Record<string, unknown>;
  const am = (o.audioMeter as Record<string, unknown>) || {};
  return {
    pending: Array.isArray(o.pending)
      ? o.pending.map((row) => {
          const r = row as Record<string, unknown>;
          const checklistRaw = r.applyChecklist;
          let applyChecklist: PanelState["pending"][number]["applyChecklist"];
          if (Array.isArray(checklistRaw)) {
            applyChecklist = checklistRaw.map((c) => {
              const x = c as Record<string, unknown>;
              const stateRaw = x.state;
              let state: DirectiveApplyChecklistRowState = "pending";
              if (
                stateRaw === "pending" ||
                stateRaw === "running" ||
                stateRaw === "done" ||
                stateRaw === "failed" ||
                stateRaw === "skipped"
              ) {
                state = stateRaw;
              }
              return {
                id:
                  typeof x.id === "string" && x.id.length > 0
                    ? x.id
                    : `check-${Math.random().toString(36).slice(2)}`,
                label: typeof x.label === "string" ? x.label : "",
                state,
                ...(typeof x.message === "string" && x.message.length > 0
                  ? { message: x.message }
                  : {}),
              };
            });
          }
          const base: PanelState["pending"][number] = {
            id: typeof r.id === "number" ? r.id : 0,
            text: typeof r.text === "string" ? r.text : "",
            receivedAt:
              typeof r.receivedAt === "string"
                ? r.receivedAt
                : new Date(0).toISOString(),
            status: r.status === "processing" ? "processing" : "queued",
          };
          if (applyChecklist !== undefined && applyChecklist.length > 0) {
            base.applyChecklist = applyChecklist;
          }
          return base;
        })
      : [],
    recentHandled: Array.isArray(o.recentHandled)
      ? o.recentHandled.map((row) => {
          const r = row as Record<string, unknown>;
          const checklistRaw = r.applyChecklist;
          let applyChecklist: PanelState["recentHandled"][number]["applyChecklist"];
          if (Array.isArray(checklistRaw)) {
            applyChecklist = checklistRaw.map((c) => {
              const x = c as Record<string, unknown>;
              const stateRaw = x.state;
              let state: DirectiveApplyChecklistRowState = "pending";
              if (
                stateRaw === "pending" ||
                stateRaw === "running" ||
                stateRaw === "done" ||
                stateRaw === "failed" ||
                stateRaw === "skipped"
              ) {
                state = stateRaw;
              }
              return {
                id:
                  typeof x.id === "string" && x.id.length > 0
                    ? x.id
                    : `done-check-${Math.random().toString(36).slice(2)}`,
                label: typeof x.label === "string" ? x.label : "",
                state,
                ...(typeof x.message === "string" && x.message.length > 0
                  ? { message: x.message }
                  : {}),
              };
            });
          }
          const base: PanelState["recentHandled"][number] = {
            text: typeof r.text === "string" ? r.text : "",
            receivedAt:
              typeof r.receivedAt === "string"
                ? r.receivedAt
                : new Date(0).toISOString(),
          };
          if (typeof r.summary === "string" && r.summary.length > 0) {
            (base as { summary?: string }).summary = r.summary;
          }
          if (typeof r.errorMessage === "string" && r.errorMessage.length > 0) {
            (base as { errorMessage?: string }).errorMessage = r.errorMessage;
          }
          if (r.skipped === true) {
            (base as { skipped?: true }).skipped = true;
          }
          if (applyChecklist !== undefined && applyChecklist.length > 0) {
            (
              base as { applyChecklist?: typeof applyChecklist }
            ).applyChecklist = applyChecklist;
          }
          return base;
        })
      : [],
    latestPartial: typeof o.latestPartial === "string" ? o.latestPartial : null,
    voiceListening: o.voiceListening === true,
    audioMeter: {
      speaking: am.speaking === true,
      rms: typeof am.rms === "number" ? am.rms : 0,
      waveform: Array.isArray(am.waveform) ? (am.waveform as number[]) : [],
    },
  };
}

export function emptyState(): PanelState {
  return {
    pending: [],
    recentHandled: [],
    latestPartial: null,
    voiceListening: false,
    audioMeter: { speaking: false, rms: 0, waveform: [] },
  };
}
