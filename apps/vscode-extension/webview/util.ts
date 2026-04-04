import type { PanelState } from "./types";

/** Last path segment for display when the host omits `preview` (Go json omitempty on empty string). */
function basenameFromPath(p: string): string {
  const s = p.replace(/\\/g, "/").trim();
  if (s === "") {
    return "";
  }
  const i = s.lastIndexOf("/");
  return i >= 0 ? s.slice(i + 1) : s;
}

function normalizeSearchStateWire(
  ss: Record<string, unknown>,
): PanelState["searchState"] | undefined {
  if (!Array.isArray(ss.results)) {
    return undefined;
  }
  const results = ss.results
    .map((r) => r as Record<string, unknown>)
    .filter((r) => typeof r.path === "string" && r.path.trim() !== "")
    .map((r) => {
      const pathStr = (r.path as string).trim();
      const rawPrev = r.preview;
      const preview =
        typeof rawPrev === "string" && rawPrev.trim() !== ""
          ? rawPrev.trim()
          : basenameFromPath(pathStr);
      const line =
        typeof r.line === "number" && Number.isFinite(r.line)
          ? r.line
          : 0;
      const character =
        typeof r.character === "number" && Number.isFinite(r.character)
          ? r.character
          : 0;
      return { path: pathStr, line, character, preview };
    });
  const activeIndex =
    typeof ss.activeIndex === "number" && Number.isFinite(ss.activeIndex)
      ? ss.activeIndex
      : 0;
  const listKind =
    ss.listKind === "file" || ss.listKind === "workspace"
      ? ss.listKind
      : undefined;
  const noHits = ss.noHits === true;
  const noHitsSummary =
    typeof ss.noHitsSummary === "string" ? ss.noHitsSummary : undefined;
  if (results.length === 0 && !noHits) {
    return undefined;
  }
  return {
    results,
    activeIndex,
    ...(listKind ? { listKind } : {}),
    ...(noHits ? { noHits: true as const } : {}),
    ...(noHitsSummary ? { noHitsSummary } : {}),
  };
}

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
      ? o.pending.map((entry) => {
          const r = entry as Record<string, unknown>;
          const pendingRow: PanelState["pending"][number] = {
            id: typeof r.id === "number" ? r.id : 0,
            text: typeof r.text === "string" ? r.text : "",
            receivedAt:
              typeof r.receivedAt === "string"
                ? r.receivedAt
                : new Date(0).toISOString(),
            status: r.status === "processing" ? "processing" : "queued",
          };
          if (
            typeof r.applyingCommandLine === "string" &&
            r.applyingCommandLine.length > 0
          ) {
            pendingRow.applyingCommandLine = r.applyingCommandLine;
          }
          if (
            typeof r.applyingCommandOutput === "string" &&
            r.applyingCommandOutput.length > 0
          ) {
            pendingRow.applyingCommandOutput = r.applyingCommandOutput;
          }
          return pendingRow;
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
  if (ss) {
    const normalized = normalizeSearchStateWire(ss);
    if (normalized) {
      base.searchState = normalized;
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
