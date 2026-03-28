import type * as vscode from "vscode";

import type { TranscriptStore } from "../voice/transcript-store";

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
        case "processing": return "Working…";
        case "error": return "Couldn’t run";
        default: return status;
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

    function drawWaveform(canvas, samples) {
      if (!canvas || !canvas.getContext) return;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      const w = canvas.width;
      const h = canvas.height;
      ctx.clearRect(0, 0, w, h);
      const fg = getComputedStyle(document.body).getPropertyValue("--vscode-textLink-foreground").trim() || "#3794ff";
      const arr = Array.isArray(samples) ? samples : [];
      if (!arr.length) return;
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

    function render(state) {
      const meterWrap = document.getElementById("meterWrap");
      const root = document.getElementById("root");
      if (!meterWrap || !root) return;

      meterWrap.innerHTML = buildMeter(state);

      const pending = Array.isArray(state.pending) ? state.pending : [];
      const recentHandled = Array.isArray(state.recentHandled)
        ? state.recentHandled
        : [];
      const voiceListening = state.voiceListening === true;
      const showLive =
        voiceListening && typeof state.latestPartial === "string" && state.latestPartial.length > 0;

      const parts = [];

      parts.push("<h1>In progress</h1>");
      if (!pending.length && !showLive) {
        if (!voiceListening) {
          parts.push('<div class="empty">Start Voice to see live transcripts here.</div>');
        } else {
          parts.push('<div class="empty">Listening — speak to see partials update here.</div>');
        }
      } else {
        parts.push('<div class="stack">');

        for (const p of pending) {
          const cls = "card pending " + p.status;
          parts.push(
            '<div class="' + esc(cls) + '">' +
              '<div class="meta">' +
                '<span class="badge">' + esc(statusLabel(p.status)) + "</span>" +
                "<span>" + esc(fmtTime(p.receivedAt)) + "</span>" +
              "</div>" +
              '<div class="text">' + esc(p.text) + "</div>" +
            "</div>"
          );
        }

        if (showLive) {
          const crumbs = (state.partialRecent || []).slice(0, -1).map(function (t) {
            return '<div class="text" style="opacity:0.65;font-size:11px;">' + esc(t) + "</div>";
          }).join("");
          parts.push(
            '<div class="card live">' +
              '<div class="meta"><span class="badge">Live</span><span>Partial (since last commit)</span></div>' +
              crumbs +
              '<div class="text">' + esc(state.latestPartial) + "</div>" +
              '<div class="typing" aria-hidden="true"><span class="dot"></span><span class="dot"></span><span class="dot"></span></div>' +
            "</div>"
          );
        }

        parts.push("</div>");
      }

      parts.push('<h1 style="margin-top:16px;">Done</h1>');
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

      parts.push('<p class="hint">Vocode · voice to code</p>');

      root.innerHTML = parts.join("");

      const cv = document.getElementById("waveCanvas");
      const am = state.audioMeter || {};
      drawWaveform(cv, am.waveform);
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
