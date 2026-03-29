import type * as vscode from "vscode";

import type { TranscriptStore } from "./transcript-store";

const viewType = "vocode.transcriptPanel";

export class TranscriptPanelViewProvider
  implements vscode.WebviewViewProvider, vscode.Disposable
{
  private view?: vscode.WebviewView;
  private readonly unsubscribe: () => void;

  constructor(
    private readonly extensionUri: vscode.Uri,
    private readonly store: TranscriptStore,
  ) {
    this.unsubscribe = this.store.onDidChange(() => {
      this.postState();
    });
  }

  resolveWebviewView(
    webviewView: vscode.WebviewView,
    _context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken,
  ): void {
    this.view = webviewView;
    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this.extensionUri],
    };
    webviewView.webview.html = this.getHtml(webviewView.webview);
    webviewView.onDidChangeVisibility(() => {
      if (webviewView.visible) {
        this.postState();
      }
    });
    this.postState();
  }

  private postState(): void {
    if (!this.view) {
      return;
    }
    const snapshot = this.store.getSnapshot();
    const plain = JSON.parse(JSON.stringify(snapshot)) as Record<
      string,
      unknown
    >;
    void this.view.webview.postMessage({
      type: "update",
      state: plain,
    });
  }

  private getHtml(webview: vscode.Webview): string {
    const csp = [
      "default-src 'none';",
      `style-src ${webview.cspSource} 'unsafe-inline';`,
      `script-src ${webview.cspSource} 'unsafe-inline';`,
    ].join(" ");

    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy" content="${csp}" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <style>
    :root {
      --pad: 12px;
      --radius: 10px;
      --gap: 10px;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      padding: var(--pad);
      font-family: var(--vscode-font-family);
      font-size: var(--vscode-font-size);
      color: var(--vscode-foreground);
      background: var(--vscode-sideBar-background);
      line-height: 1.45;
    }
    h1 {
      margin: 0 0 var(--gap);
      font-size: 11px;
      font-weight: 600;
      letter-spacing: 0.04em;
      text-transform: uppercase;
      color: var(--vscode-descriptionForeground);
    }
    .stack { display: flex; flex-direction: column; gap: var(--gap); }
    .card {
      border-radius: var(--radius);
      padding: 10px 12px;
      background: var(--vscode-input-background);
      border: 1px solid var(--vscode-widget-border, rgba(128,128,128,0.25));
      box-shadow: 0 1px 2px rgba(0,0,0,0.06);
    }
    .card.done { opacity: 0.85; }
    .card.summary {
      border-left: 3px solid var(--vscode-notificationsInfoIcon-foreground, var(--vscode-textLink-foreground));
      background: var(--vscode-editor-inactiveSelectionBackground, var(--vscode-input-background));
    }
    .summary-for {
      font-size: 11px;
      color: var(--vscode-descriptionForeground);
      margin-top: 8px;
      white-space: pre-wrap;
      word-break: break-word;
    }
    .error-detail {
      margin-top: 8px;
      font-size: 11px;
      color: var(--vscode-errorForeground);
      white-space: pre-wrap;
      word-break: break-word;
    }
    .card.pending { border-left: 3px solid var(--vscode-progressBar-background); }
    .card.processing { border-left: 3px solid var(--vscode-textLink-foreground); }
    .card.error { border-left: 3px solid var(--vscode-errorForeground); }
    .meta {
      font-size: 10px;
      color: var(--vscode-descriptionForeground);
      margin-bottom: 6px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .badge {
      font-size: 10px;
      padding: 2px 6px;
      border-radius: 999px;
      background: var(--vscode-badge-background);
      color: var(--vscode-badge-foreground);
    }
    .text { white-space: pre-wrap; word-break: break-word; }
    .live {
      border-style: dashed;
      background: var(--vscode-editor-inactiveSelectionBackground, var(--vscode-input-background));
    }
    .live .typing {
      display: inline-flex;
      gap: 4px;
      align-items: center;
      margin-top: 4px;
    }
    .dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--vscode-textLink-foreground);
      opacity: 0.85;
      animation: pulse 1.2s ease-in-out infinite;
    }
    .dot:nth-child(2) { animation-delay: 0.15s; }
    .dot:nth-child(3) { animation-delay: 0.3s; }
    @keyframes pulse {
      0%, 100% { opacity: 0.35; transform: scale(0.9); }
      50% { opacity: 1; transform: scale(1); }
    }
    .hint {
      font-size: 11px;
      color: var(--vscode-descriptionForeground);
      margin-top: 8px;
      text-align: center;
    }
    .empty {
      padding: 24px 12px;
      text-align: center;
      color: var(--vscode-descriptionForeground);
      font-size: 12px;
    }
    .meter { margin-bottom: 14px; }
    .meter-bar {
      height: 8px;
      border-radius: 4px;
      background: var(--vscode-input-background);
      overflow: hidden;
      border: 1px solid var(--vscode-widget-border, rgba(128,128,128,0.2));
    }
    .meter-fill {
      height: 100%;
      background: var(--vscode-textLink-foreground);
      transition: width 0.07s ease-out;
      border-radius: 3px;
    }
    .wave-canvas {
      width: 100%;
      max-width: 100%;
      height: 44px;
      display: block;
      margin-top: 8px;
      border-radius: 6px;
      background: var(--vscode-editor-inactiveSelectionBackground, var(--vscode-input-background));
    }
  </style>
</head>
<body>
  <div id="meterWrap"></div>
  <div id="root"></div>
  <script>
    const vscode = acquireVsCodeApi();

    function esc(s) {
      const d = document.createElement("div");
      d.textContent = s;
      return d.innerHTML;
    }

    function fmtTime(d) {
      try {
        return new Intl.DateTimeFormat(undefined, {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        }).format(new Date(d));
      } catch {
        return "";
      }
    }

    function statusLabel(status) {
      switch (status) {
        case "queued": return "Queued";
        case "processing": return "Applying…";
        case "error": return "Couldn’t run";
        default: return status;
      }
    }

    function statusBadgeTitle(status) {
      switch (status) {
        case "queued":
          return "Committed transcript — waiting to run in the workspace";
        case "processing":
          return "Running this committed line through the Vocode agent";
        case "error":
          return "Something went wrong applying this line";
        default:
          return "";
      }
    }

    function buildMeter(state) {
      const voiceListening = state.voiceListening === true;
      const am = state.audioMeter || {};
      const rms = typeof am.rms === "number" ? am.rms : 0;
      const speaking = am.speaking === true;
      const pct = Math.round(Math.min(1, Math.max(0, rms)) * 100);
      if (!voiceListening) {
        return (
          '<div class="meter card">' +
            '<div class="meta"><span class="badge">Mic</span><span>Idle</span></div>' +
            '<div class="empty" style="padding:10px 8px;font-size:11px;">Start Voice for level, VAD, and waveform.</div>' +
          "</div>"
        );
      }
      return (
        '<div class="meter card">' +
          '<div class="meta">' +
            '<span class="badge">' + esc(speaking ? "Speaking" : "Quiet") + "</span>" +
            "<span>Input level</span>" +
          "</div>" +
          '<div class="meter-bar"><div class="meter-fill" style="width:' + pct + '%"></div></div>' +
          '<canvas id="waveCanvas" class="wave-canvas" width="320" height="44" aria-label="Recent level"></canvas>' +
        "</div>"
      );
    }

    /** Stable key for #root; when only audio_meter changes, skip replacing #root so live-card CSS animations are not reset ~25/s. */
    function mainContentSignature(state) {
      return JSON.stringify({
        v: state.voiceListening === true,
        lp: typeof state.latestPartial === "string" ? state.latestPartial : "",
        p: Array.isArray(state.pending) ? state.pending : [],
        r: Array.isArray(state.recentHandled) ? state.recentHandled : [],
      });
    }

    var lastAppliedMainSig = null;
    var mainDebounceTimer = null;
    var pendingMainState = null;

    function drawWaveform(canvas, samples) {
      if (!canvas || !canvas.getContext) return;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      const w = canvas.width;
      const h = canvas.height;
      ctx.clearRect(0, 0, w, h);
      const fg = getComputedStyle(document.body).getPropertyValue("--vscode-textLink-foreground").trim() || "#3794ff";
      const arr = Array.isArray(samples) ? samples : [];
      if (!arr.length) {
        ctx.strokeStyle = fg;
        ctx.globalAlpha = 0.22;
        ctx.beginPath();
        ctx.moveTo(0, h - 2);
        ctx.lineTo(w, h - 2);
        ctx.stroke();
        ctx.globalAlpha = 1;
        return;
      }
      const n = arr.length;
      const gap = 1;
      const barW = Math.max(1, (w - (n - 1) * gap) / n);
      let x = 0;
      for (let i = 0; i < n; i++) {
        const v = typeof arr[i] === "number" ? arr[i] : 0;
        const bh = Math.max(1, Math.min(1, v) * (h - 4));
        ctx.fillStyle = fg;
        ctx.globalAlpha = 0.75 + 0.25 * Math.min(1, v);
        ctx.fillRect(x, h - bh, barW, bh);
        x += barW + gap;
      }
      ctx.globalAlpha = 1;
    }

    function buildRootHtml(state) {
      const pending = Array.isArray(state.pending) ? state.pending : [];
      const recentHandled = Array.isArray(state.recentHandled)
        ? state.recentHandled
        : [];
      const voiceListening = state.voiceListening === true;
      const showLive =
        voiceListening && typeof state.latestPartial === "string" && state.latestPartial.length > 0;

      const parts = [];
      const sectionTitleMargin = ' style="margin-top:16px;"';

      parts.push("<h1>Live</h1>");
      if (!voiceListening) {
        parts.push(
          '<div class="empty">Start Voice to stream speech-to-text here.</div>'
        );
      } else if (!showLive) {
        parts.push(
          '<div class="empty">No draft yet — speak to see streaming text.</div>'
        );
      } else {
        parts.push('<div class="stack">');
        parts.push(
          '<div class="card live">' +
            '<div class="meta">' +
            '<span class="badge" title="Streaming speech-to-text — not final until it moves to Done">Live</span>' +
            '<span title="Draft before the provider commits this segment">Draft</span>' +
            "</div>" +
            '<div class="text">' + esc(state.latestPartial) + "</div>" +
            '<div class="typing" aria-hidden="true"><span class="dot"></span><span class="dot"></span><span class="dot"></span></div>' +
            "</div>"
        );
        parts.push("</div>");
      }

      parts.push("<h1" + sectionTitleMargin + ">Applying</h1>");
      if (!pending.length) {
        parts.push(
          '<div class="empty">Nothing running on the workspace from voice yet.</div>'
        );
      } else {
        parts.push('<div class="stack">');
        for (const p of pending) {
          const cls = "card pending " + p.status;
          const bt = statusBadgeTitle(p.status);
          const badge =
            '<span class="badge"' +
            (bt ? ' title="' + esc(bt) + '"' : "") +
            ">" +
            esc(statusLabel(p.status)) +
            "</span>";
          parts.push(
            '<div class="' + esc(cls) + '">' +
              '<div class="meta">' +
                badge +
                "<span>" + esc(fmtTime(p.receivedAt)) + "</span>" +
              "</div>" +
              '<div class="text">' + esc(p.text) + "</div>" +
              (p.status === "error" && typeof p.errorMessage === "string" && p.errorMessage.length > 0
                ? '<div class="error-detail">Error: ' + esc(p.errorMessage) + "</div>"
                : "") +
            "</div>"
          );
        }
        parts.push("</div>");
      }

      parts.push("<h1" + sectionTitleMargin + ">Done</h1>");
      if (!recentHandled.length) {
        parts.push('<div class="empty">Completed requests will appear here.</div>');
      } else {
        parts.push('<div class="stack">');
        for (const h of recentHandled) {
          parts.push(
            '<div class="card done">' +
              '<div class="meta"><span>' + esc(fmtTime(h.receivedAt)) + "</span></div>" +
              '<div class="text">' + esc(h.text) + "</div>" +
            "</div>"
          );
        }
        parts.push("</div>");
      }

      parts.push("<h1" + sectionTitleMargin + ">Summary</h1>");
      const withSummaries = recentHandled.filter(
        function (h) {
          return typeof h.summary === "string" && h.summary.length > 0;
        },
      );
      if (!withSummaries.length) {
        parts.push(
          '<div class="empty">When the planner finishes with a short summary, it appears here with the matching transcript.</div>',
        );
      } else {
        parts.push('<div class="stack">');
        for (const h of withSummaries) {
          const preview =
            typeof h.text === "string" && h.text.length > 140
              ? h.text.slice(0, 140) + "…"
              : h.text || "";
          parts.push(
            '<div class="card summary">' +
              '<div class="meta">' +
              '<span class="badge" title="Planner done summary for this turn">Summary</span>' +
              "<span>" + esc(fmtTime(h.receivedAt)) + "</span>" +
              "</div>" +
              '<div class="text">' + esc(h.summary) + "</div>" +
              (preview
                ? '<div class="summary-for">Transcript: ' + esc(preview) + "</div>"
                : "") +
              "</div>",
          );
        }
        parts.push("</div>");
      }

      parts.push('<p class="hint">Vocode · voice to code</p>');

      return parts.join("");
    }

    function flushMainDebounce() {
      if (mainDebounceTimer !== null) {
        clearTimeout(mainDebounceTimer);
        mainDebounceTimer = null;
      }
      pendingMainState = null;
    }

    function render(state) {
      const meterWrap = document.getElementById("meterWrap");
      const root = document.getElementById("root");
      if (!meterWrap || !root) return;

      meterWrap.innerHTML = buildMeter(state);
      const cv0 = document.getElementById("waveCanvas");
      drawWaveform(cv0, (state.audioMeter || {}).waveform);

      const sig = mainContentSignature(state);
      if (sig === lastAppliedMainSig) {
        flushMainDebounce();
        return;
      }

      if (
        pendingMainState !== null &&
        mainContentSignature(pendingMainState) === sig
      ) {
        return;
      }

      pendingMainState = state;
      if (mainDebounceTimer !== null) {
        clearTimeout(mainDebounceTimer);
      }
      var delay = lastAppliedMainSig === null ? 0 : 45;
      mainDebounceTimer = setTimeout(function () {
        mainDebounceTimer = null;
        var s = pendingMainState;
        pendingMainState = null;
        if (!s) return;
        var finalSig = mainContentSignature(s);
        if (finalSig === lastAppliedMainSig) return;
        root.innerHTML = buildRootHtml(s);
        lastAppliedMainSig = finalSig;
        var cv = document.getElementById("waveCanvas");
        drawWaveform(cv, (s.audioMeter || {}).waveform);
      }, delay);
    }

    window.addEventListener("message", function (event) {
      const msg = event.data;
      if (msg && msg.type === "update" && msg.state) {
        render(msg.state);
      }
    });
  </script>
</body>
</html>`;
  }

  dispose(): void {
    this.unsubscribe();
  }
}

export const transcriptPanelViewType = viewType;
