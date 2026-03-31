import type {
  DirectiveApplyChecklistRowState,
  HandledRow,
  PanelState,
  PendingRow,
} from "./types";
import { fmtTime, statusBadgeTitle, statusLabel } from "./util";

function LiveSection({ state }: { state: PanelState }) {
  const voiceListening = state.voiceListening === true;
  const partial =
    typeof state.latestPartial === "string" && state.latestPartial.length > 0
      ? state.latestPartial
      : null;
  const showLive = voiceListening && partial !== null;

  return (
    <>
      <h1>Live</h1>
      {!voiceListening ? (
        <div className="empty" />
      ) : !showLive ? (
        <div className="empty" />
      ) : (
        <div className="stack">
          <div className="card live">
            <div className="meta">
              <span
                className="badge"
                title="Streaming speech-to-text — not final until it moves below"
              >
                Live
              </span>
              <span title="Draft before the provider commits this segment">
                Draft
              </span>
            </div>
            <div className="text">{partial}</div>
            <div className="typing" aria-hidden="true">
              <span className="dot" />
              <span className="dot" />
              <span className="dot" />
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function ClarifySection({ state }: { state: PanelState }) {
  const prompt = state.clarifyPrompt;
  if (!prompt || !prompt.question) {
    return null;
  }
  return (
    <>
      <h1>Clarification needed</h1>
      <div className="stack">
        <div className="card done failed history-card">
          <div className="meta">
            <span className="badge" title="Vocode needs one detail to proceed">
              Question
            </span>
          </div>
          <div className="text history-summary">{prompt.question}</div>
          <div className="history-transcript muted-transcript">
            Original: {prompt.originalTranscript}
          </div>
          <div className="hint">
            Speak your answer next — Vocode will treat the next utterance as the
            reply.
          </div>
        </div>
      </div>
    </>
  );
}

function AnswerSection({ state }: { state: PanelState }) {
  const ans = state.answerState;
  if (!ans || !ans.answerText) {
    return null;
  }
  return (
    <>
      <h1>Answer</h1>
      <div className="stack">
        <div className="card history-card">
          <div className="meta">
            <span className="badge" title="AI answer (no editor actions)">
              Answer
            </span>
          </div>
          <div className="text history-summary">{ans.answerText}</div>
          <div className="history-transcript muted-transcript">
            Question: {ans.question}
          </div>
        </div>
      </div>
    </>
  );
}

type ApplyStepVisual =
  | "done"
  | "active"
  | "pending"
  | "failed"
  | "directive-skipped";

function applyPipelineSteps(status: PendingRow["status"]): {
  label: string;
  visual: ApplyStepVisual;
  title?: string;
}[] {
  const st = status;
  return [
    { label: "Transcript committed", visual: "done" },
    {
      label: "Run agent",
      visual: st === "processing" ? "active" : "pending",
      title:
        st === "queued"
          ? "Waiting to send this line to the daemon"
          : "Agent loop is running",
    },
  ];
}

function checklistRowVisual(
  state: DirectiveApplyChecklistRowState,
): ApplyStepVisual {
  switch (state) {
    case "done":
      return "done";
    case "running":
      return "active";
    case "failed":
      return "failed";
    case "skipped":
      return "directive-skipped";
    default:
      return "pending";
  }
}

function ApplyStepRow({
  label,
  visual,
  title,
}: {
  label: string;
  visual: ApplyStepVisual;
  title?: string;
}) {
  return (
    <div className={`apply-step apply-step-${visual}`} title={title}>
      <span className="apply-step-mark" aria-hidden="true" />
      <span className="apply-step-label">{label}</span>
    </div>
  );
}

function DirectiveChecklist({
  items,
  showTitle = true,
}: {
  items: readonly {
    id: string;
    label: string;
    state: DirectiveApplyChecklistRowState;
    message?: string;
  }[];
  showTitle?: boolean;
}) {
  return (
    <>
      {showTitle ? <h2 className="apply-steps-subhead">Directives</h2> : null}
      <div
        className="apply-steps apply-steps-directives"
        role="list"
        aria-label="Directives to apply"
      >
        {items.map((item) => (
          <ApplyStepRow
            key={item.id}
            label={item.label}
            visual={checklistRowVisual(item.state)}
            title={
              item.message !== undefined && item.message.length > 0
                ? item.message
                : undefined
            }
          />
        ))}
      </div>
    </>
  );
}

function CompactQueuedCard({ p }: { p: PendingRow }) {
  return (
    <div className={`card pending-compact pending ${p.status}`}>
      <div className="meta">
        <span className="badge" title={statusBadgeTitle(p.status) || undefined}>
          {statusLabel(p.status)}
        </span>
        <span>{fmtTime(p.receivedAt)}</span>
      </div>
      <div className="text pending-compact-text">{p.text}</div>
    </div>
  );
}

function ApplyingSection({ pending }: { pending: readonly PendingRow[] }) {
  const primary = pending[0];
  const queuedRest = pending.length > 1 ? pending.slice(1) : [];

  return (
    <>
      <h1 className="section-title">Applying</h1>
      {!primary ? (
        <div className="empty" />
      ) : (
        <div className="stack">
          <div className={`card pending ${primary.status}`}>
            <div className="meta">
              <span
                className="badge"
                title={statusBadgeTitle(primary.status) || undefined}
              >
                {statusLabel(primary.status)}
              </span>
              <span>{fmtTime(primary.receivedAt)}</span>
            </div>
            <div className="text">{primary.text}</div>
            <div className="apply-steps" role="list" aria-label="Pipeline">
              {applyPipelineSteps(primary.status).map((s) => (
                <ApplyStepRow key={s.label} {...s} />
              ))}
            </div>
            {primary.applyChecklist !== undefined &&
            primary.applyChecklist.length > 0 ? (
              <DirectiveChecklist items={primary.applyChecklist} />
            ) : null}
          </div>
          {queuedRest.map((p) => (
            <CompactQueuedCard key={p.id} p={p} />
          ))}
        </div>
      )}
    </>
  );
}

function HistoryCard({ h }: { h: HandledRow }) {
  const failed =
    typeof h.errorMessage === "string" && h.errorMessage.length > 0;
  const summary =
    typeof h.summary === "string" && h.summary.trim().length > 0
      ? h.summary.trim()
      : null;
  const checklist =
    h.applyChecklist !== undefined && h.applyChecklist.length > 0
      ? h.applyChecklist
      : null;
  const checklistCounts =
    checklist === null
      ? null
      : checklist.reduce(
          (acc, item) => {
            switch (item.state) {
              case "done":
                acc.done += 1;
                break;
              case "failed":
                acc.failed += 1;
                break;
              case "skipped":
                acc.skipped += 1;
                break;
              case "running":
                acc.running += 1;
                break;
              default:
                acc.pending += 1;
            }
            return acc;
          },
          { done: 0, failed: 0, skipped: 0, running: 0, pending: 0 },
        );

  if (failed) {
    return (
      <div className="card done failed history-card">
        <div className="meta">
          <span
            className="badge"
            title="Daemon or workspace apply did not succeed"
          >
            {"Couldn't run"}
          </span>
          <span>{fmtTime(h.receivedAt)}</span>
        </div>
        {summary ? <div className="text history-summary">{summary}</div> : null}
        <div className="history-transcript muted-transcript">{h.text}</div>
        <div className="error-detail">Error: {h.errorMessage}</div>
        {checklist !== null ? (
          <details className="history-directives">
            <summary className="history-directives-summary">
              <span className="history-directives-summary-label">
                Directives ({checklist.length})
              </span>
              {checklistCounts ? (
                <span className="history-directives-summary-counts">
                  {checklistCounts.done} done
                  {checklistCounts.failed > 0
                    ? ` • ${checklistCounts.failed} failed`
                    : ""}
                </span>
              ) : null}
            </summary>
            <DirectiveChecklist items={checklist} showTitle={false} />
          </details>
        ) : null}
      </div>
    );
  }

  return (
    <div className="card done history-card">
      <div className="meta">
        <span>{fmtTime(h.receivedAt)}</span>
      </div>
      {summary ? (
        <>
          <div className="text history-summary">{summary}</div>
          <div className="history-transcript muted-transcript">{h.text}</div>
        </>
      ) : (
        <div className="text">{h.text}</div>
      )}
      {checklist !== null ? (
        <details className="history-directives">
          <summary className="history-directives-summary">
            <span className="history-directives-summary-label">
              Directives ({checklist.length})
            </span>
            {checklistCounts ? (
              <span className="history-directives-summary-counts">
                {checklistCounts.done} done
                {checklistCounts.failed > 0
                  ? ` • ${checklistCounts.failed} failed`
                  : ""}
              </span>
            ) : null}
          </summary>
          <DirectiveChecklist items={checklist} showTitle={false} />
        </details>
      ) : null}
    </div>
  );
}

function HistorySection({ items }: { items: readonly HandledRow[] }) {
  return (
    <>
      <h1 className="section-title">Recent</h1>
      {!items.length ? (
        <div className="empty" />
      ) : (
        <div className="stack">
          {items.map((h) => (
            <HistoryCard key={`h-${h.receivedAt}-${h.text}`} h={h} />
          ))}
        </div>
      )}
    </>
  );
}

function SkippedCard({ h }: { h: HandledRow }) {
  const summary =
    typeof h.summary === "string" && h.summary.trim().length > 0
      ? h.summary.trim()
      : null;
  return (
    <div className="card skipped-card">
      <div className="meta">
        <span
          className="badge"
          title="Agent treated this line as not actionable"
        >
          Skipped
        </span>
        <span>{fmtTime(h.receivedAt)}</span>
      </div>
      {summary ? (
        <>
          <div className="text skipped-summary">{summary}</div>
          <div className="history-transcript muted-transcript">{h.text}</div>
        </>
      ) : (
        <div className="text muted-transcript">{h.text}</div>
      )}
    </div>
  );
}

function SkippedSection({ items }: { items: readonly HandledRow[] }) {
  if (!items.length) {
    return null;
  }
  return (
    <>
      <h1 className="section-title">Skipped</h1>
      <div className="stack">
        {items.map((h) => (
          <SkippedCard key={`s-${h.receivedAt}-${h.text}`} h={h} />
        ))}
      </div>
    </>
  );
}

function ChatSection({ state }: { state: PanelState }) {
  const handled = Array.isArray(state.recentHandled) ? state.recentHandled : [];
  const items = handled
    .filter(
      (h) => h.transcriptOutcome === "answer" && !!(h.answerText || h.summary),
    )
    .map((h) => ({
      question: h.text,
      answerText: h.answerText || h.summary || "",
      receivedAt: h.receivedAt,
    }));
  return (
    <>
      <h1>Chat</h1>
      <div className="stack">
        {items.length === 0 ? <div className="empty" /> : null}
        {items.map((qa) => (
          <div
            key={`qa-${qa.receivedAt}-${qa.question}`}
            className="card history-card"
          >
            <div className="meta">
              <span className="badge" title="Question">
                Q
              </span>
              <span className="muted-transcript">{qa.receivedAt}</span>
            </div>
            <div className="text">{qa.question}</div>
            <div className="meta" style={{ marginTop: 8 }}>
              <span className="badge" title="Answer">
                A
              </span>
            </div>
            <div className="text history-summary">{qa.answerText}</div>
          </div>
        ))}
      </div>
    </>
  );
}

function SearchResultsSection({ state }: { state: PanelState }) {
  const ss = state.searchState;
  if (!ss || !Array.isArray(ss.results) || ss.results.length === 0) {
    return null;
  }
  const active = Math.min(
    Math.max(0, Number.isFinite(ss.activeIndex) ? ss.activeIndex : 0),
    ss.results.length - 1,
  );
  return (
    <>
      <h1>Search results</h1>
      <div className="stack">
        {ss.results.map((r, i) => (
          <div
            key={`sr-${r.path}:${r.line}:${r.character}`}
            className={`card history-card ${i === active ? "card-active" : ""}`}
          >
            <div className="meta">
              <span className="badge" title="Result number for voice selection">
                {i + 1}
              </span>
              <span className="muted-transcript">
                {r.path}:{r.line + 1}:{r.character + 1}
              </span>
            </div>
            <div className="text mono">{r.preview}</div>
            {i === active ? (
              <div className="hint">
                Active. Say “next”, “back”, or a number (e.g. “3”) to jump.
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </>
  );
}

export function MainPanel({ state }: { state: PanelState }) {
  const pending = Array.isArray(state.pending) ? state.pending : [];
  const recentHandled = Array.isArray(state.recentHandled)
    ? state.recentHandled
    : [];
  const skippedItems = recentHandled.filter((h) => h.skipped === true);
  // Q/A belongs in Chat; keep Recent focused on coding actions.
  const historyItems = recentHandled.filter(
    (h) => h.skipped !== true && h.transcriptOutcome !== "answer",
  );

  return (
    <div id="main-root">
      <ChatSection state={state} />
      <ClarifySection state={state} />
      <SearchResultsSection state={state} />
      <LiveSection state={state} />
      <AnswerSection state={state} />
      <ApplyingSection pending={pending} />
      <HistorySection items={historyItems} />
      <SkippedSection items={skippedItems} />
      <p className="hint">Vocode · Speak code</p>
    </div>
  );
}
