import { randomUUID } from "node:crypto";

/**
 * Observable state for the main voice sidebar webview (Live / Applying / Recent / Skipped,
 * partial hypotheses, listening flag, audio meter). Daemon transcript application lives in
 * ../transcript/apply-directives — this store is UI-only.
 */
const DEFAULT_MAX_HANDLED = 30;
const WAVEFORM_SAMPLES = 64;
/** STT sometimes sends empty/whitespace partials between words; debounce clearing so the Live card does not flicker off. */
const DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS = 180;

/** Committed voice text not yet finished processing through the daemon + apply pipeline. */
export type PendingStatus = "queued" | "processing";

export type DirectiveApplyChecklistState =
  | "pending"
  | "running"
  | "done"
  | "failed"
  | "skipped";

export type DirectiveApplyChecklistItem = {
  readonly id: string;
  readonly label: string;
  state: DirectiveApplyChecklistState;
  message?: string;
};

export type PendingTranscript = {
  readonly id: number;
  readonly text: string;
  readonly receivedAt: Date;
  status: PendingStatus;
  /** Filled while `host.applyDirectives` runs for this voice line (duplex apply). */
  applyChecklist?: DirectiveApplyChecklistItem[];
};

export type MainPanelSnapshot = {
  /** Committed lines still in flight (queued, processing, or error). */
  readonly pending: readonly PendingTranscript[];
  /**
   * Recently finished lines (newest first): success (optional `summary` for Summary panel)
   * or failure (`errorMessage` when daemon/apply did not complete successfully).
   */
  readonly recentHandled: readonly {
    readonly text: string;
    readonly receivedAt: Date;
    readonly summary?: string;
    readonly errorMessage?: string;
    /** Agent marked transcript as irrelevant / non-actionable; listed in the Skipped section. */
    readonly skipped?: true;
    /** Final directive checklist captured from Applying for user-visible history. */
    readonly applyChecklist?: readonly DirectiveApplyChecklistItem[];
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
    applyChecklist?: readonly DirectiveApplyChecklistItem[];
  }[] = [];

  private latestPartial: string | null = null;
  private voiceListening = false;

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

  markProcessing(id: number): void {
    const item = this.pending.find((p) => p.id === id);
    if (item) {
      item.status = "processing";
      this.emit();
    }
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

  /** Number of directive rows already on this pending line (across prior apply batches / repair rounds). */
  directiveApplyChecklistLength(pendingId: number): number {
    const item = this.pending.find((p) => p.id === pendingId);
    return item?.applyChecklist?.length ?? 0;
  }

  /**
   * Appends rows for a new host apply batch. Prior rows (e.g. completed repair rounds) stay visible.
   */
  appendDirectiveApplyChecklist(
    pendingId: number,
    labels: readonly string[],
  ): void {
    const item = this.pending.find((p) => p.id === pendingId);
    if (!item || labels.length === 0) {
      return;
    }
    if (!item.applyChecklist) {
      item.applyChecklist = [];
    }
    for (const label of labels) {
      item.applyChecklist.push({
        id: randomUUID(),
        label,
        state: "pending",
      });
    }
    this.emit();
  }

  setDirectiveApplyItemState(
    pendingId: number,
    index: number,
    state: DirectiveApplyChecklistState,
    message?: string,
  ): void {
    const item = this.pending.find((p) => p.id === pendingId);
    const list = item?.applyChecklist;
    if (!list || index < 0 || index >= list.length) {
      return;
    }
    const row = list[index];
    if (!row) {
      return;
    }
    row.state = state;
    if (message !== undefined && message.trim() !== "") {
      row.message = message.trim();
    } else if (state !== "failed") {
      delete row.message;
    }
    this.emit();
  }

  markHandled(
    id: number,
    options?: {
      summary?: string;
      transcriptOutcome?: "irrelevant" | "completed";
    },
  ): void {
    const index = this.pending.findIndex((p) => p.id === id);
    if (index === -1) {
      return;
    }
    const [removed] = this.pending.splice(index, 1);
    const summary = options?.summary?.trim();
    const skipped =
      options?.transcriptOutcome === "irrelevant" ? (true as const) : undefined;
    this.recentHandled.unshift({
      text: removed.text,
      receivedAt: removed.receivedAt,
      ...(summary ? { summary } : {}),
      ...(skipped ? { skipped } : {}),
      ...(removed.applyChecklist !== undefined &&
      removed.applyChecklist.length > 0
        ? {
            applyChecklist: removed.applyChecklist.map((item) => ({ ...item })),
          }
        : {}),
    });
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  /**
   * Records a completed transcript that did not go through the pending queue (e.g. manual send).
   * Shown under Done; optional summary appears in the Summary section; optional errorMessage shows a failed card.
   */
  recordCompletedTranscript(
    text: string,
    options?: {
      summary?: string;
      errorMessage?: string;
      transcriptOutcome?: "irrelevant" | "completed";
    },
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
        : options?.transcriptOutcome === "irrelevant"
          ? (true as const)
          : undefined;
    this.recentHandled.unshift({
      text: normalized,
      receivedAt: new Date(),
      ...(err !== undefined && err !== ""
        ? { errorMessage: err }
        : summary
          ? { summary }
          : {}),
      ...(skipped ? { skipped } : {}),
    });
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
      ...(removed.applyChecklist !== undefined &&
      removed.applyChecklist.length > 0
        ? {
            applyChecklist: removed.applyChecklist.map((item) => ({ ...item })),
          }
        : {}),
    });
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  getSnapshot(): MainPanelSnapshot {
    return {
      pending: this.pending,
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
