export type TranscriptKind = "partial" | "final";

export type TranscriptEntry = {
  readonly id: number;
  readonly text: string;
  readonly kind: TranscriptKind;
  readonly receivedAt: Date;
};

const DEFAULT_MAX_ENTRIES = 50;

export class TranscriptStore {
  private readonly entries: TranscriptEntry[] = [];
  private readonly listeners = new Set<() => void>();
  private nextId = 1;

  constructor(private readonly maxEntries: number = DEFAULT_MAX_ENTRIES) {}

  add(text: string, kind: TranscriptKind): TranscriptEntry {
    const normalizedText = text.trim();
    const entry: TranscriptEntry = {
      id: this.nextId,
      text: normalizedText,
      kind,
      receivedAt: new Date(),
    };

    this.nextId++;

    if (!normalizedText) {
      return entry;
    }

    this.entries.unshift(entry);
    if (this.entries.length > this.maxEntries) {
      this.entries.pop();
    }

    this.emit();
    return entry;
  }

  getEntries(): readonly TranscriptEntry[] {
    return this.entries;
  }

  onDidChange(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  private emit(): void {
    for (const listener of this.listeners) {
      listener();
    }
  }
}
