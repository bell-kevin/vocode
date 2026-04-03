import { getVsCodeApi } from "../../api/vscode";
import type { PanelState } from "../../types";

export function SearchPanel({ state }: { state: PanelState }) {
  const ss = state.searchState;
  if (!ss || !Array.isArray(ss.results) || ss.results.length === 0) {
    return null;
  }
  const active = Math.min(
    Math.max(0, Number.isFinite(ss.activeIndex) ? ss.activeIndex : 0),
    ss.results.length - 1,
  );
  const isFile = ss.listKind === "file";
  const cancelControl = isFile ? "cancel_file_selection" : "cancel_selection";
  const kicker = isFile
    ? `${ss.results.length} file${ss.results.length === 1 ? "" : "s"} — the highlighted row is active in the editor.`
    : `${ss.results.length} match${ss.results.length === 1 ? "" : "es"} — the highlighted row is active in the editor.`;

  return (
    <div className="interrupt-panel">
      <p className="interrupt-panel-kicker">{kicker}</p>
      <div className="stack interrupt-panel-results">
        {ss.results.map((r, i) => (
          <div
            key={`sr-${r.path}:${r.line}:${r.character}`}
            className={`card history-card interrupt-search-card ${
              i === active ? "card-active" : ""
            }`}
          >
            <div className="meta">
              <span className="badge" title="Say this number to jump">
                {i + 1}
              </span>
              <span className="muted-transcript">
                {isFile
                  ? r.path
                  : `${r.path}:${r.line + 1}:${r.character + 1}`}
              </span>
            </div>
            <div className="text mono interrupt-search-preview">
              {r.preview}
            </div>
            {i === active ? (
              <div className="hint interrupt-search-hint">
                Active result. Say “next”, “back”, or a number (e.g. “3”) to
                move selection.
              </div>
            ) : null}
          </div>
        ))}
      </div>
      <p className="interrupt-panel-footnote">
        Voice navigation stays active until you close this list.
      </p>
      <div className="interrupt-actions">
        <button
          type="button"
          className="interrupt-secondary-btn"
          onClick={() =>
            getVsCodeApi()?.postMessage({
              type: "transcriptControl",
              control: cancelControl,
            })
          }
        >
          Close results
        </button>
      </div>
    </div>
  );
}
