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

export type DirectiveApplyChecklistRowState =
  | "pending"
  | "running"
  | "done"
  | "failed"
  | "skipped";

export type PendingRow = {
  id: number;
  text: string;
  receivedAt: string;
  status: "queued" | "processing";
  applyChecklist?: readonly {
    id: string;
    label: string;
    state: DirectiveApplyChecklistRowState;
    message?: string;
  }[];
};

export type HandledRow = {
  text: string;
  receivedAt: string;
  summary?: string;
  transcriptOutcome?: "irrelevant" | "completed" | "clarify" | "search" | "answer";
  answerText?: string;
  errorMessage?: string;
  /** Irrelevant / non-actionable transcript (daemon transcriptOutcome irrelevant). */
  skipped?: true;
  applyChecklist?: readonly {
    id: string;
    label: string;
    state: DirectiveApplyChecklistRowState;
    message?: string;
  }[];
};
