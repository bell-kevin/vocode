import path from "node:path";
import type {
  VoiceTranscriptClarifyOffer,
  VoiceTranscriptFileListHit,
  VoiceTranscriptFileSearchState,
  VoiceTranscriptQuestionAnswer,
  VoiceTranscriptWorkspaceHints,
  VoiceTranscriptWorkspaceSearchState,
} from "@vocode/protocol";

/**
 * Observable state for the main voice sidebar webview (Live / Applying / Recent / Skipped,
 * partial hypotheses, listening flag, audio meter). Daemon transcript application lives in
 * ../../voice-transcript/apply-directives — this store is UI-only.
 */
const DEFAULT_MAX_HANDLED = 30;
const WAVEFORM_SAMPLES = 64;
/** STT sometimes sends empty/whitespace partials between words; debounce clearing so the Live card does not flicker off. */
const DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS = 180;

/** Mirrors daemon inferTranscriptUIDisposition when completion omits uiDisposition. */
function inferVoiceTranscriptUiDisposition(opts: {
  search?: VoiceTranscriptWorkspaceSearchState;
  question?: VoiceTranscriptQuestionAnswer;
  clarify?: VoiceTranscriptClarifyOffer;
  fileSelection?: VoiceTranscriptFileSearchState;
  workspace?: VoiceTranscriptWorkspaceHints;
}): "shown" | "skipped" | "hidden" | "browse" {
  const ans = opts.question?.answerText?.trim();
  if (ans) {
    return "hidden";
  }
  const s = opts.search;
  if (s) {
    if (s.closed || s.noHits || (s.results && s.results.length > 0)) {
      return "browse";
    }
  }
  if (
    opts.clarify &&
    typeof opts.clarify.targetResolution === "string" &&
    opts.clarify.targetResolution.trim() !== ""
  ) {
    return "hidden";
  }
  if (opts.fileSelection) {
    return "browse";
  }
  if (opts.workspace?.needsFolder === true) {
    return "shown";
  }
  return "shown";
}

/**
 * `uiDisposition: "browse"` means Search/file-list side UI only — exclude successful lines from main panel History.
 */
function shouldLogTranscriptHistoryEntry(args: {
  filledAnswer: boolean;
  disp: "shown" | "skipped" | "hidden" | "browse";
  summary?: string;
  errorMessage?: string;
  options?: TranscriptHandledOptions;
}): boolean {
  if (args.filledAnswer) {
    return false;
  }
  const err = args.errorMessage?.trim();
  if (err) {
    return true;
  }
  if (args.disp === "browse") {
    return false;
  }
  if (args.disp === "skipped" || args.disp === "shown") {
    return true;
  }
  return !!args.summary?.trim();
}

function sidebarRowsFromFileHits(hits: readonly VoiceTranscriptFileListHit[]): {
  path: string;
  line: number;
  character: number;
  preview: string;
}[] {
  return hits.map((h) => ({
    path: h.path,
    line: 0,
    character: 0,
    preview: (h.preview?.trim() || path.basename(h.path)) ?? "",
  }));
}

export type TranscriptHandledOptions = {
  summary?: string;
  uiDisposition?: "shown" | "skipped" | "hidden" | "browse";
  contextSessionId?: string;
  search?: VoiceTranscriptWorkspaceSearchState;
  question?: VoiceTranscriptQuestionAnswer;
  clarify?: VoiceTranscriptClarifyOffer;
  fileSelection?: VoiceTranscriptFileSearchState;
  workspace?: VoiceTranscriptWorkspaceHints;
};

/** Committed voice text not yet finished processing through the daemon + apply pipeline. */
export type PendingStatus = "queued" | "processing";

/** Cap stored command output so the webview stays responsive for noisy CLIs. */
const maxApplyingCommandOutputChars = 96_000;

export type PendingTranscript = {
  readonly id: number;
  readonly text: string;
  readonly receivedAt: Date;
  status: PendingStatus;
  /** Set while the host runs a shell directive for this row (voice transcript RPC). */
  applyingCommandLine?: string;
  applyingCommandOutput?: string;
};

export type MainPanelSnapshot = {
  /** Committed lines still in flight (queued, processing, or error). */
  readonly pending: readonly PendingTranscript[];
  /** When set, the daemon requested clarification and voice input should answer this question. */
  readonly clarifyPrompt?: {
    readonly question: string;
    readonly originalTranscript: string;
  };
  /** Latest search hit list + active selection (voice "next/back/3" updates this). */
  readonly searchState?: {
    readonly results: readonly {
      readonly path: string;
      readonly line: number;
      readonly character: number;
      readonly preview: string;
    }[];
    readonly activeIndex: number;
    /** Workspace text hits vs file path hits (same sidebar list UX). */
    readonly listKind?: "workspace" | "file";
    /** Search / file path ran but returned no hits — show empty-state in Search panel. */
    readonly noHits?: boolean;
    readonly noHitsSummary?: string;
  };
  /** Latest question answer (`VoiceTranscriptCompletion.question`). */
  readonly answerState?: {
    readonly question: string;
    readonly answerText: string;
  };
  /** Recent Q/A pairs (newest first). */
  readonly qaHistory?: readonly {
    readonly question: string;
    readonly answerText: string;
    readonly receivedAt: Date;
  }[];
  /**
   * Recently finished lines (newest first): mutations, skips, prompts — not routine search/browse
   * (those stay in the Search panel). All errors (`errorMessage`) are always listed.
   */
  readonly recentHandled: readonly {
    readonly text: string;
    readonly receivedAt: Date;
    readonly summary?: string;
    readonly errorMessage?: string;
    /** Agent marked transcript as irrelevant / non-actionable; listed in the Skipped section. */
    readonly skipped?: true;
  }[];
  /** Latest partial hypothesis after the most recent committed event. */
  readonly latestPartial: string | null;
  /** True while Start Voice is active (matches extension voice session). */
  readonly voiceListening: boolean;
  /** Sidecar VAD + level (streaming STT only); for waveform / level UI. */
  readonly audioMeter: AudioMeterSnapshot;
};

export type AudioMeterSnapshot = {
  readonly speaking: boolean;
  /** Normalized 0-1 RMS level from sidecar. */
  readonly rms: number;
  /** Recent normalized levels (oldest → newest) for a simple waveform strip. */
  readonly waveform: readonly number[];
};

export class MainPanelStore {
  private readonly listeners = new Set<() => void>();
  private nextId = 1;

  private readonly pending: PendingTranscript[] = [];
  private recentHandled: {
    text: string;
    receivedAt: Date;
    summary?: string;
    errorMessage?: string;
    skipped?: true;
  }[] = [];

  private latestPartial: string | null = null;
  private voiceListening = false;

  private clarifyPrompt:
    | {
        question: string;
        originalTranscript: string;
        /** Daemon session key when this prompt was shown; used for cancel_clarify RPC. */
        contextSessionId?: string;
      }
    | undefined;

  private searchState:
    | {
        results: readonly {
          path: string;
          line: number;
          character: number;
          preview: string;
        }[];
        activeIndex: number;
        listKind?: "workspace" | "file";
        noHits?: boolean;
        noHitsSummary?: string;
        /** Daemon session key for this hit list; used for cancel_* RPC after voice stops. */
        contextSessionId?: string;
      }
    | undefined;

  private answerState:
    | {
        question: string;
        answerText: string;
      }
    | undefined;

  private qaHistory: {
    question: string;
    answerText: string;
    receivedAt: Date;
  }[] = [];

  private meterSpeaking = false;
  private meterRms = 0;
  private waveformRms: number[] = [];

  private partialClearTimer: ReturnType<typeof setTimeout> | undefined;

  /**
   * FIFO queue of voice `pending` ids currently awaiting `voice.transcript` (daemon may call
   * `host.applyDirectives` during the first entry). Used to attribute apply batches to the sidebar row.
   */
  private voiceTranscriptRpcOrder: number[] = [];

  constructor(
    private readonly maxHandled: number = DEFAULT_MAX_HANDLED,
    private readonly partialClearDebounceMs: number = DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS,
  ) {}

  /**
   * Whether the user has started voice listening. The panel only shows live partials + typing dots
   * when this is true and a partial exists (extension has no VAD; partials proxy “speech in flight”).
   */
  setVoiceListening(active: boolean): void {
    if (this.voiceListening === active) {
      return;
    }
    this.voiceListening = active;
    if (!active) {
      this.clearPartialClearTimer();
      this.latestPartial = null;
      this.meterSpeaking = false;
      this.meterRms = 0;
      this.waveformRms = [];
    } else {
      this.clearPartialClearTimer();
      this.meterSpeaking = false;
      this.meterRms = 0;
      this.waveformRms = [];
    }
    this.emit();
  }

  /** Mic level + VAD speech gate from sidecar `audio_meter` events (streaming only). */
  setAudioMeter(speaking: boolean, rms: number): void {
    if (!this.voiceListening) {
      return;
    }
    const level = Number.isFinite(rms) ? Math.min(1, Math.max(0, rms)) : 0;
    this.meterSpeaking = speaking;
    this.meterRms = level;
    this.waveformRms.push(level);
    if (this.waveformRms.length > WAVEFORM_SAMPLES) {
      this.waveformRms.shift();
    }
    this.emit();
  }

  /** Partial hypotheses after the latest committed transcript (cleared on each commit). */
  onPartial(text: string): void {
    if (!this.voiceListening) {
      return;
    }
    const normalized = text.trim();
    if (!normalized) {
      if (this.partialClearDebounceMs <= 0) {
        this.latestPartial = null;
        this.clearPartialClearTimer();
        this.emit();
        return;
      }
      this.clearPartialClearTimer();
      this.partialClearTimer = setTimeout(() => {
        this.partialClearTimer = undefined;
        if (!this.voiceListening) {
          return;
        }
        this.latestPartial = null;
        this.emit();
      }, this.partialClearDebounceMs);
      return;
    }

    this.clearPartialClearTimer();
    this.latestPartial = normalized;
    this.emit();
  }

  /**
   * A committed line arrived from STT. Clears the partial buffer for the next utterance.
   * Returns the id used to track this line through the daemon pipeline.
   */
  /** Returns null if the committed text was empty (nothing to track). */
  enqueueCommitted(text: string): number | null {
    const normalized = text.trim();
    this.clearPartialClearTimer();
    this.latestPartial = null;

    if (!normalized) {
      this.emit();
      return null;
    }

    const id = this.nextId++;
    this.pending.push({
      id,
      text: normalized,
      receivedAt: new Date(),
      status: "queued",
    });
    this.emit();
    return id;
  }

  /**
   * If a clarification prompt is active, consumes it and returns a combined text to send to the daemon.
   * The returned string is the transcript text to send; UI still displays the committed line normally.
   */
  /**
   * User aborted clarification: clear the prompt and record the original line under Skipped
   * so the flow has an explicit outcome (not a silent dismiss).
   */
  abortClarifyAsSkipped(): void {
    if (!this.clarifyPrompt) {
      return;
    }
    const { question, originalTranscript } = this.clarifyPrompt;
    this.clarifyPrompt = undefined;
    const text = originalTranscript.trim() || "(empty transcript)";
    const q = question.trim();
    const shortQ = q.length > 160 ? `${q.slice(0, 157)}…` : q;
    this.recentHandled.unshift({
      text,
      receivedAt: new Date(),
      skipped: true,
      summary: `Clarification cancelled. Question was: ${shortQ}`,
    });
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  /** Clear the search hit list from the sidebar (user closed the search panel). */
  dismissSearchState(): void {
    if (!this.searchState) {
      return;
    }
    this.searchState = undefined;
    this.emit();
  }

  /** Opaque daemon `contextSessionId` for the active clarify prompt, if known. */
  clarifyPromptContextSessionId(): string | undefined {
    return this.clarifyPrompt?.contextSessionId;
  }

  /** Opaque daemon `contextSessionId` tied to the current search hit list, if known. */
  searchContextSessionId(): string | undefined {
    return this.searchState?.contextSessionId;
  }

  consumeClarifyPromptAnswer(answerText: string): string | null {
    if (!this.clarifyPrompt) {
      return null;
    }
    const answer = answerText.trim();
    if (!answer) {
      return null;
    }
    const { question, originalTranscript } = this.clarifyPrompt;
    // Clear immediately so subsequent utterances are treated normally.
    this.clarifyPrompt = undefined;
    this.emit();
    return [
      originalTranscript.trim(),
      "",
      `Clarifying question: ${question.trim()}`,
      `User answer: ${answer}`,
    ].join("\n");
  }

  /**
   * Consumes clarify prompt and returns both the daemon text to send and the UI text to display
   * for the committed utterance (we attribute completion to the original instruction).
   */
  consumeClarifyPromptAnswerForSend(
    answerText: string,
  ): { sendText: string; displayText: string } | null {
    if (!this.clarifyPrompt) {
      return null;
    }
    const answer = answerText.trim();
    if (!answer) {
      return null;
    }
    const { question, originalTranscript } = this.clarifyPrompt;
    // Clear immediately so subsequent utterances are treated normally.
    this.clarifyPrompt = undefined;
    this.emit();
    return {
      displayText: originalTranscript.trim() || "Clarification",
      sendText: [
        originalTranscript.trim(),
        "",
        `Clarifying question: ${question.trim()}`,
        `User answer: ${answer}`,
      ].join("\n"),
    };
  }

  markProcessing(id: number): void {
    const item = this.pending.find((p) => p.id === id);
    if (item) {
      item.status = "processing";
      this.emit();
    }
  }

  /**
   * Shows the shell invocation line in the Applying card while `host.applyDirectives` runs a command.
   */
  setApplyingCommandLine(id: number, commandLine: string): void {
    const item = this.pending.find((p) => p.id === id);
    if (!item) {
      return;
    }
    const line = commandLine.trim();
    item.applyingCommandLine = line !== "" ? line : undefined;
    item.applyingCommandOutput = "";
    this.emit();
  }

  /** Appends streamed stdout/stderr (already formatted) for the Applying card. */
  appendApplyingCommandOutput(id: number, chunk: string): void {
    if (chunk === "") {
      return;
    }
    const item = this.pending.find((p) => p.id === id);
    if (!item) {
      return;
    }
    let next = (item.applyingCommandOutput ?? "") + chunk;
    if (next.length > maxApplyingCommandOutputChars) {
      const marker = "\n…(earlier output truncated)…\n";
      next =
        marker +
        next.slice(next.length - maxApplyingCommandOutputChars + marker.length);
    }
    item.applyingCommandOutput = next;
    this.emit();
  }

  /** Call immediately before `client.transcript` for a voice-committed line. */
  beginVoiceTranscriptRpc(pendingId: number): void {
    this.voiceTranscriptRpcOrder.push(pendingId);
  }

  /** Call in `finally` after `client.transcript` settles for that line. */
  endVoiceTranscriptRpc(pendingId: number): void {
    const i = this.voiceTranscriptRpcOrder.indexOf(pendingId);
    if (i >= 0) {
      this.voiceTranscriptRpcOrder.splice(i, 1);
    }
  }

  /** Pending row id for the voice transcript RPC the daemon is executing (front of FIFO). */
  activeVoiceTranscriptRpcPendingId(): number | undefined {
    return this.voiceTranscriptRpcOrder[0];
  }

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: intentionally exhaustive state reducer
  markHandled(id: number, options?: TranscriptHandledOptions): void {
    const index = this.pending.findIndex((p) => p.id === id);
    if (index === -1) {
      return;
    }
    const [removed] = this.pending.splice(index, 1);
    const summary = options?.summary?.trim();
    const skipped =
      options?.uiDisposition === "skipped" ? (true as const) : undefined;
    if (options?.clarify?.targetResolution?.trim() && summary) {
      this.clarifyPrompt = {
        question: summary,
        originalTranscript: removed.text,
        ...(options.contextSessionId
          ? { contextSessionId: options.contextSessionId }
          : {}),
      };
      // Clarify is an in-progress flow; do not add the original utterance to history yet.
      while (this.recentHandled.length > this.maxHandled) {
        this.recentHandled.pop();
      }
      this.emit();
      return;
    }
    if (options?.search) {
      const s = options.search;
      if (s.closed) {
        this.searchState = undefined;
      } else if (s.noHits) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: [],
          activeIndex: 0,
          listKind: "workspace",
          noHits: true,
          noHitsSummary:
            summary && summary !== "" ? summary : "No matches for this search.",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      } else if (s.results && s.results.length > 0) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: s.results,
          activeIndex: Math.max(0, s.activeIndex ?? 0),
          listKind: "workspace",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      }
    }

    if (options?.fileSelection) {
      const f = options.fileSelection;
      if (f.closed) {
        this.searchState = undefined;
      } else if (f.noHits) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: [],
          activeIndex: 0,
          listKind: "file",
          noHits: true,
          noHitsSummary:
            summary && summary !== ""
              ? summary
              : "No file paths matched this search.",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      } else if (f.results && f.results.length > 0) {
        const prevCtx = this.searchState?.contextSessionId;
        const ai = Math.max(0, f.activeIndex ?? 0);
        const capped = Math.min(ai, f.results.length - 1);
        this.searchState = {
          results: sidebarRowsFromFileHits(f.results),
          activeIndex: capped,
          listKind: "file",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      }
    }

    let filledAnswer = false;
    if (options?.question) {
      const ans =
        options.question.answerText?.trim() ??
        (summary !== undefined && summary !== "" ? summary : undefined);
      if (ans) {
        filledAnswer = true;
        this.answerState = { question: removed.text, answerText: ans };
        this.qaHistory.unshift({
          question: removed.text,
          answerText: ans,
          receivedAt: removed.receivedAt,
        });
        while (this.qaHistory.length > this.maxHandled) {
          this.qaHistory.pop();
        }
      }
    }

    const disp: "shown" | "skipped" | "hidden" | "browse" =
      options?.uiDisposition ??
      inferVoiceTranscriptUiDisposition({
        search: options?.search,
        question: options?.question,
        clarify: options?.clarify,
        fileSelection: options?.fileSelection,
        workspace: options?.workspace,
      });

    // Don't put answers into Recent — they belong in Chat.
    const shouldLogToRecent = shouldLogTranscriptHistoryEntry({
      filledAnswer,
      disp,
      summary,
      options,
    });
    if (shouldLogToRecent) {
      this.recentHandled.unshift({
        text: removed.text,
        receivedAt: removed.receivedAt,
        ...(summary ? { summary } : {}),
        ...(disp === "skipped" || skipped ? { skipped: true as const } : {}),
      });
    }
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  /**
   * Records a completed transcript without a pending row (tests and rare callers).
   * Shown under Done; optional summary appears in the Summary section; optional errorMessage shows a failed card.
   */

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: intentionally exhaustive state reducer
  recordCompletedTranscript(
    text: string,
    options?: TranscriptHandledOptions & { errorMessage?: string },
  ): void {
    const normalized = text.trim();
    if (!normalized) {
      return;
    }
    const summary = options?.summary?.trim();
    const err = options?.errorMessage?.trim();
    const skipped =
      err !== undefined && err !== ""
        ? undefined
        : options?.uiDisposition === "skipped"
          ? (true as const)
          : undefined;
    if (options?.search) {
      const s = options.search;
      if (s.closed) {
        this.searchState = undefined;
      } else if (s.noHits) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: [],
          activeIndex: 0,
          listKind: "workspace",
          noHits: true,
          noHitsSummary:
            summary && summary !== "" ? summary : "No matches for this search.",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      } else if (s.results && s.results.length > 0) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: s.results,
          activeIndex: Math.max(0, s.activeIndex ?? 0),
          listKind: "workspace",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      }
    }

    if (options?.fileSelection) {
      const f = options.fileSelection;
      if (f.closed) {
        this.searchState = undefined;
      } else if (f.noHits) {
        const prevCtx = this.searchState?.contextSessionId;
        this.searchState = {
          results: [],
          activeIndex: 0,
          listKind: "file",
          noHits: true,
          noHitsSummary:
            summary && summary !== ""
              ? summary
              : "No file paths matched this search.",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      } else if (f.results && f.results.length > 0) {
        const prevCtx = this.searchState?.contextSessionId;
        const ai = Math.max(0, f.activeIndex ?? 0);
        const capped = Math.min(ai, f.results.length - 1);
        this.searchState = {
          results: sidebarRowsFromFileHits(f.results),
          activeIndex: capped,
          listKind: "file",
          contextSessionId: options.contextSessionId ?? prevCtx,
        };
      }
    }

    let filledAnswer = false;
    if (options?.question) {
      const ans =
        options.question.answerText?.trim() ??
        (summary !== undefined && summary !== "" ? summary : undefined);
      if (ans) {
        filledAnswer = true;
        this.answerState = { question: normalized, answerText: ans };
        this.qaHistory.unshift({
          question: normalized,
          answerText: ans,
          receivedAt: new Date(),
        });
        while (this.qaHistory.length > this.maxHandled) {
          this.qaHistory.pop();
        }
      }
    }

    const disp: "shown" | "skipped" | "hidden" | "browse" =
      options?.uiDisposition ??
      inferVoiceTranscriptUiDisposition({
        search: options?.search,
        question: options?.question,
        clarify: options?.clarify,
        fileSelection: options?.fileSelection,
        workspace: options?.workspace,
      });

    // Don't put answers into Recent — they belong in Chat.
    const shouldLogToRecent = shouldLogTranscriptHistoryEntry({
      filledAnswer,
      disp,
      summary,
      errorMessage: err,
      options,
    });
    if (shouldLogToRecent) {
      this.recentHandled.unshift({
        text: normalized,
        receivedAt: new Date(),
        ...(err !== undefined && err !== ""
          ? { errorMessage: err }
          : summary
            ? { summary }
            : {}),
        ...(disp === "skipped" || skipped ? { skipped: true as const } : {}),
      });
    }
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  /**
   * Finishes a pending line as failed: removes it from Applying and appends to Done with error detail.
   */
  markError(id: number, errorMessage?: string): void {
    const index = this.pending.findIndex((p) => p.id === id);
    if (index === -1) {
      return;
    }
    const [removed] = this.pending.splice(index, 1);
    const err = errorMessage?.trim() || undefined;
    this.recentHandled.unshift({
      text: removed.text,
      receivedAt: removed.receivedAt,
      ...(err !== undefined && err !== "" ? { errorMessage: err } : {}),
    });
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  getSnapshot(): MainPanelSnapshot {
    return {
      pending: this.pending,
      ...(this.clarifyPrompt
        ? {
            clarifyPrompt: {
              question: this.clarifyPrompt.question,
              originalTranscript: this.clarifyPrompt.originalTranscript,
            },
          }
        : {}),
      ...(this.searchState
        ? {
            searchState: {
              results: this.searchState.results,
              activeIndex: this.searchState.activeIndex,
              ...(this.searchState.listKind
                ? { listKind: this.searchState.listKind }
                : {}),
              ...(this.searchState.noHits ? { noHits: true as const } : {}),
              ...(this.searchState.noHitsSummary
                ? { noHitsSummary: this.searchState.noHitsSummary }
                : {}),
            },
          }
        : {}),
      ...(this.answerState ? { answerState: this.answerState } : {}),
      ...(this.qaHistory.length > 0 ? { qaHistory: this.qaHistory } : {}),
      recentHandled: this.recentHandled,
      latestPartial: this.latestPartial,
      voiceListening: this.voiceListening,
      audioMeter: {
        speaking: this.meterSpeaking,
        rms: this.meterRms,
        waveform: this.waveformRms,
      },
    };
  }

  onDidChange(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  private emit(): void {
    for (const listener of this.listeners) {
      try {
        listener();
      } catch (err) {
        console.error("[MainPanelStore] listener threw:", err);
      }
    }
  }

  private clearPartialClearTimer(): void {
    if (this.partialClearTimer !== undefined) {
      clearTimeout(this.partialClearTimer);
      this.partialClearTimer = undefined;
    }
  }
}
