import type { PanelState } from "./types";

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
      ? (o.pending as PanelState["pending"])
      : [],
    recentHandled: Array.isArray(o.recentHandled)
      ? (o.recentHandled as PanelState["recentHandled"])
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
