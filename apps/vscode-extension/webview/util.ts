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
  const base: PanelState = {
    pending: Array.isArray(o.pending)
      ? o.pending.map((row) => {
          const r = row as Record<string, unknown>;
          return {
            id: typeof r.id === "number" ? r.id : 0,
            text: typeof r.text === "string" ? r.text : "",
            receivedAt:
              typeof r.receivedAt === "string"
                ? r.receivedAt
                : new Date(0).toISOString(),
            status: r.status === "processing" ? "processing" : "queued",
          };
        })
      : [],
    recentHandled: Array.isArray(o.recentHandled)
      ? o.recentHandled.map((row) => {
          const r = row as Record<string, unknown>;
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

  const cp = o.clarifyPrompt as Record<string, unknown> | undefined;
  if (
    cp &&
    typeof cp.question === "string" &&
    typeof cp.originalTranscript === "string"
  ) {
    base.clarifyPrompt = {
      question: cp.question,
      originalTranscript: cp.originalTranscript,
    };
  }

  const ss = o.searchState as Record<string, unknown> | undefined;
  if (ss && Array.isArray(ss.results)) {
    const results = ss.results
      .map((r) => r as Record<string, unknown>)
      .filter(
        (r) =>
          typeof r.path === "string" &&
          typeof r.line === "number" &&
          typeof r.character === "number" &&
          typeof r.preview === "string",
      )
      .map((r) => ({
        path: r.path as string,
        line: r.line as number,
        character: r.character as number,
        preview: r.preview as string,
      }));
    if (results.length > 0) {
      const activeIndex =
        typeof ss.activeIndex === "number" && Number.isFinite(ss.activeIndex)
          ? ss.activeIndex
          : 0;
      const listKind =
        ss.listKind === "file" || ss.listKind === "workspace"
          ? ss.listKind
          : undefined;
      base.searchState = {
        results,
        activeIndex,
        ...(listKind ? { listKind } : {}),
      };
    }
  }

  const as = o.answerState as Record<string, unknown> | undefined;
  if (
    as &&
    typeof as.question === "string" &&
    typeof as.answerText === "string"
  ) {
    base.answerState = { question: as.question, answerText: as.answerText };
  }

  const qh = o.qaHistory;
  if (Array.isArray(qh)) {
    base.qaHistory = qh
      .map((x) => x as Record<string, unknown>)
      .filter(
        (x) =>
          typeof x.question === "string" &&
          typeof x.answerText === "string" &&
          typeof x.receivedAt === "string",
      )
      .map((x) => ({
        question: x.question as string,
        answerText: x.answerText as string,
        receivedAt: x.receivedAt as string,
      }));
  }

  return base;
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
