/** Wire shape after JSON postMessage (dates are ISO strings). */
export type PanelState = {
  pending: readonly PendingRow[];
  clarifyPrompt?: {
    question: string;
    originalTranscript: string;
  };
  searchState?: {
    results: readonly {
      path: string;
      line: number;
      character: number;
      preview: string;
    }[];
    activeIndex: number;
  };
  answerState?: {
    question: string;
    answerText: string;
  };
  qaHistory?: readonly {
    question: string;
    answerText: string;
    receivedAt: string;
  }[];
  recentHandled: readonly HandledRow[];
  latestPartial: string | null;
  voiceListening: boolean;
  audioMeter: {
    speaking: boolean;
    rms: number;
    waveform: readonly number[];
  };
};

export type PendingRow = {
  id: number;
  text: string;
  receivedAt: string;
  status: "queued" | "processing";
};

export type HandledRow = {
  text: string;
  receivedAt: string;
  summary?: string;
  errorMessage?: string;
  /** Irrelevant / non-actionable transcript (`uiDisposition: skipped`). */
  skipped?: true;
};
