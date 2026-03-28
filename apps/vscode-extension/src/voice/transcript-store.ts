const DEFAULT_MAX_HANDLED = 30;
const WAVEFORM_SAMPLES = 64;
/** STT sometimes sends empty/whitespace partials between words; debounce clearing so the Live card does not flicker off. */
const DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS = 180;

/** Committed voice text not yet finished processing through the daemon + apply pipeline. */
export type PendingStatus = "queued" | "processing" | "error";

export type PendingTranscript = {
  readonly id: number;
  readonly text: string;
  readonly receivedAt: Date;
  status: PendingStatus;
};

export type TranscriptPanelSnapshot = {
  /** Committed lines still in flight (queued, processing, or error). */
  readonly pending: readonly PendingTranscript[];
  /** Recently completed lines (newest first). */
  readonly recentHandled: readonly {
    readonly text: string;
    readonly receivedAt: Date;
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
  /** Normalized 0–1 RMS level from sidecar. */
  readonly rms: number;
  /** Recent normalized levels (oldest → newest) for a simple waveform strip. */
  readonly waveform: readonly number[];
};

export class TranscriptStore {
  private readonly listeners = new Set<() => void>();
  private nextId = 1;

  private readonly pending: PendingTranscript[] = [];
  private recentHandled: { text: string; receivedAt: Date }[] = [];

  private latestPartial: string | null = null;
  private voiceListening = false;

  private meterSpeaking = false;
  private meterRms = 0;
  private waveformRms: number[] = [];

  private partialClearTimer: ReturnType<typeof setTimeout> | undefined;

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

  markHandled(id: number): void {
    const index = this.pending.findIndex((p) => p.id === id);
    if (index === -1) {
      return;
    }
    const [removed] = this.pending.splice(index, 1);
    this.recentHandled.unshift({
      text: removed.text,
      receivedAt: removed.receivedAt,
    });
    while (this.recentHandled.length > this.maxHandled) {
      this.recentHandled.pop();
    }
    this.emit();
  }

  markError(id: number): void {
    const item = this.pending.find((p) => p.id === id);
    if (item) {
      item.status = "error";
      this.emit();
    }
  }

  getSnapshot(): TranscriptPanelSnapshot {
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
        console.error("[TranscriptStore] listener threw:", err);
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
