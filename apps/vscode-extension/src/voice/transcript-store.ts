const DEFAULT_MAX_HANDLED = 30;
const WAVEFORM_SAMPLES = 64;
/** STT sometimes sends empty/whitespace partials between words; debounce clearing so the Live card does not flicker off. */
const DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS = 180;
/** Cosmetic only: if streaming partials oscillate (e.g. till/until), clear Live after silence. Authoritative text is always committed lines from the sidecar. */
const PARTIAL_FLIPFLOP_SILENCE_CLEAR_MS = 4500;

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

  /** Last few non-empty partial strings (for A/B/A/B oscillation detection). */
  private partialRecent: string[] = [];
  /** True once we detected alternating between two hypotheses (STT streaming artifact). */
  private partialFlipFlopActive = false;
  /** Last time VAD reported speech (audio_meter); used to clear stuck flip-flop partials after silence. */
  private lastAudibleAt = 0;

  constructor(
    private readonly maxHandled: number = DEFAULT_MAX_HANDLED,
    private readonly partialClearDebounceMs: number = DEFAULT_PARTIAL_CLEAR_DEBOUNCE_MS,
    private readonly flipFlopSilenceClearMs: number = PARTIAL_FLIPFLOP_SILENCE_CLEAR_MS,
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
      this.resetPartialStreamingState();
    } else {
      this.clearPartialClearTimer();
      this.meterSpeaking = false;
      this.meterRms = 0;
      this.waveformRms = [];
      this.resetPartialStreamingState();
      this.lastAudibleAt = Date.now();
    }
    this.emit();
  }

  /** Mic level + VAD speech gate from sidecar `audio_meter` events (streaming only). */
  setAudioMeter(speaking: boolean, rms: number): void {
    if (!this.voiceListening) {
      return;
    }
    const level = Number.isFinite(rms) ? Math.min(1, Math.max(0, rms)) : 0;
    if (speaking) {
      this.lastAudibleAt = Date.now();
    } else if (
      this.latestPartial &&
      this.partialFlipFlopActive &&
      Date.now() - this.lastAudibleAt >= this.flipFlopSilenceClearMs
    ) {
      this.resetPartialStreamingState();
      this.latestPartial = null;
      this.clearPartialClearTimer();
      this.meterSpeaking = speaking;
      this.meterRms = level;
      this.waveformRms.push(level);
      if (this.waveformRms.length > WAVEFORM_SAMPLES) {
        this.waveformRms.shift();
      }
      this.emit();
      return;
    }
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
        this.resetPartialStreamingState();
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
        this.resetPartialStreamingState();
        this.emit();
      }, this.partialClearDebounceMs);
      return;
    }

    this.clearPartialClearTimer();
    this.partialRecent.push(normalized);
    if (this.partialRecent.length > 4) {
      this.partialRecent.shift();
    }
    this.latestPartial = this.stabilizePartialDisplay(normalized);
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
    this.resetPartialStreamingState();
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

  private resetPartialStreamingState(): void {
    this.partialRecent = [];
    this.partialFlipFlopActive = false;
  }

  /**
   * ElevenLabs often re-hypothesizes the same tail audio (e.g. "till" vs "until") on successive
   * partial_transcript events. When we see A,B,A,B, freeze UI to a stable string instead of flickering.
   */
  private stabilizePartialDisplay(latest: string): string {
    const r = this.partialRecent;
    if (r.length >= 4) {
      const a = r[r.length - 4] as string;
      const b = r[r.length - 3] as string;
      const c = r[r.length - 2] as string;
      const d = r[r.length - 1] as string;
      if (a === c && b === d && a !== b) {
        this.partialFlipFlopActive = true;
        if (a.length !== b.length) {
          return a.length > b.length ? a : b;
        }
        return a;
      }
    }
    return latest;
  }
}
