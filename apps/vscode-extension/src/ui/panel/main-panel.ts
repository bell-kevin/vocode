import * as vscode from "vscode";

import type { ExtensionServices } from "../../commands/services";
import {
  applyVocodePanelConfigPatch,
  buildVocodePanelConfigMessage,
} from "../../config/panel-settings";
import {
  ANTHROPIC_API_KEY_SECRET,
  ELEVENLABS_API_KEY_SECRET,
  getVocodeSetupBlockReason,
  OPENAI_API_KEY_SECRET,
} from "../../config/spawn-env";
import { collapseNonemptySelectionsInActiveEditor } from "../../voice-transcript/collapse-selection";
import { sendTranscriptControlRequest } from "../../voice-transcript/transcript-control";
import {
  type MainPanelStore,
  panelHadActiveSearchInterrupt,
} from "./main-panel-store";

/** VS Code contributed view id (package.json); stable for user layouts and commands. */
const mainPanelViewId = "vocode.transcriptPanel";

/**
 * Webview provider for the extension’s main sidebar panel (voice transcript UI).
 * Loads the React bundle from `dist/webview/main-panel.{js,css}`.
 */
export class MainPanelViewProvider
  implements vscode.WebviewViewProvider, vscode.Disposable
{
  private view?: vscode.WebviewView;
  private pendingOpenPanelView: "settings" | "main" | null = null;
  private readonly unsubscribe: () => void;
  private readonly voiceStatusDisposable: vscode.Disposable;
  /** Previous snapshot: whether search/file interrupt was showing (for collapsing editor selection when flow ends). */
  private lastHadSearchInterrupt: boolean;

  constructor(
    private readonly extensionUri: vscode.Uri,
    private readonly store: MainPanelStore,
    private readonly extensionContext: vscode.ExtensionContext,
    private readonly services: ExtensionServices,
  ) {
    this.lastHadSearchInterrupt = panelHadActiveSearchInterrupt(
      this.store.getSnapshot(),
    );
    this.unsubscribe = this.store.onDidChange(() => {
      const snap = this.store.getSnapshot();
      const now = panelHadActiveSearchInterrupt(snap);
      if (this.lastHadSearchInterrupt && !now) {
        collapseNonemptySelectionsInActiveEditor();
      }
      this.lastHadSearchInterrupt = now;
      this.postState();
    });
    this.voiceStatusDisposable = this.services.voiceStatus.onDidChangeState(
      () => {
        this.postVoiceUiStatus();
      },
    );
  }

  /** Focuses the sidebar view and switches the webview to Settings (or Main). */
  revealPanelView(panelView: "settings" | "main"): void {
    this.pendingOpenPanelView = panelView;
    void vscode.commands
      .executeCommand(`${mainPanelViewId}.focus`)
      .then(() => this.flushPendingPanelView());
  }

  private flushPendingPanelView(): void {
    const v = this.pendingOpenPanelView;
    if (v === null) {
      return;
    }
    if (!this.view) {
      return;
    }
    this.pendingOpenPanelView = null;
    void this.view.webview.postMessage({ type: "openPanelView", panelView: v });
  }

  private onWebviewMessage(wv: vscode.Webview, msg: unknown): void {
    if (!msg || typeof msg !== "object") {
      return;
    }
    const m = msg as Record<string, unknown>;
    const secretSetByType: Record<string, string> = {
      setElevenLabsApiKey: ELEVENLABS_API_KEY_SECRET,
      setOpenAIApiKey: OPENAI_API_KEY_SECRET,
      setAnthropicApiKey: ANTHROPIC_API_KEY_SECRET,
    };
    if (m.type === "webviewReady") {
      void (async () => {
        const block = await getVocodeSetupBlockReason(this.extensionContext);
        void wv.postMessage({
          type: "initialRoute",
          panelView: block === null ? "main" : "settings",
        });
        void wv.postMessage(
          await buildVocodePanelConfigMessage(this.extensionContext),
        );
        void wv.postMessage({
          type: "voiceUiStatus",
          state: this.services.voiceStatus.getState(),
        });
        this.flushPendingPanelView();
      })();
      return;
    }
    if (m.type === "toggleVoiceUiStatus") {
      void (async () => {
        const vs = this.services.voiceStatus;
        if (vs.getState() === "idle") {
          await vscode.commands.executeCommand("vocode.startVoice");
        } else {
          await vscode.commands.executeCommand("vocode.stopVoice");
        }
        // Panel label updates via `voiceStatus.onDidChangeState`.
      })();
      return;
    }
    if (m.type === "requestPanelConfig") {
      void (async () => {
        void wv.postMessage(
          await buildVocodePanelConfigMessage(this.extensionContext),
        );
      })();
      return;
    }
    if (typeof m.type === "string" && m.type in secretSetByType) {
      const v = m.value;
      const s = typeof v === "string" ? v.trim() : "";
      const secretKey = secretSetByType[m.type];
      void (async () => {
        try {
          if (s === "") {
            await this.extensionContext.secrets.delete(secretKey);
          } else {
            await this.extensionContext.secrets.store(secretKey, s);
          }
        } finally {
          void wv.postMessage(
            await buildVocodePanelConfigMessage(this.extensionContext),
          );
        }
      })();
      return;
    }
    if (m.type === "setPanelConfig") {
      const patch = m.patch;
      if (!patch || typeof patch !== "object") {
        return;
      }
      void (async () => {
        try {
          await applyVocodePanelConfigPatch(patch as Record<string, unknown>);
        } finally {
          void wv.postMessage(
            await buildVocodePanelConfigMessage(this.extensionContext),
          );
        }
      })();
      return;
    }
    if (m.type === "transcriptControl") {
      const c = m.control;
      if (
        c !== "cancel_clarify" &&
        c !== "cancel_selection" &&
        c !== "cancel_file_selection"
      ) {
        return;
      }
      void (async () => {
        const ctx =
          c === "cancel_clarify"
            ? this.store.clarifyPromptContextSessionId()
            : this.store.searchContextSessionId();
        const ok = await sendTranscriptControlRequest(
          this.services,
          c,
          ctx ?? this.services.voiceSession.contextSessionId(),
        );
        if (!ok) {
          return;
        }
        if (c === "cancel_clarify") {
          this.store.abortClarifyAsSkipped();
        } else {
          this.store.dismissSearchState();
        }
      })();
      return;
    }
    if (m.type === "openExtensionSettings") {
      void vscode.commands.executeCommand(
        "workbench.action.openSettings",
        "vocode",
      );
      return;
    }
    if (m.type === "restartVocodeBackend") {
      // No longer used (settings apply automatically).
    }
  }

  resolveWebviewView(
    webviewView: vscode.WebviewView,
    _context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken,
  ): void {
    this.view = webviewView;
    const webviewRoot = vscode.Uri.joinPath(
      this.extensionUri,
      "dist",
      "webview",
    );
    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this.extensionUri],
    };
    webviewView.webview.html = this.getHtml(webviewView.webview, webviewRoot);
    webviewView.onDidChangeVisibility(() => {
      if (webviewView.visible) {
        this.postState();
      }
    });
    this.postState();
    this.flushPendingPanelView();

    const wv = webviewView.webview;
    const disposables: vscode.Disposable[] = [];

    disposables.push(
      wv.onDidReceiveMessage((msg: unknown) => {
        this.onWebviewMessage(wv, msg);
      }),
    );

    disposables.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration("vocode")) {
          void (async () => {
            void wv.postMessage(
              await buildVocodePanelConfigMessage(this.extensionContext),
            );
          })();
        }
      }),
    );

    disposables.push(
      this.extensionContext.secrets.onDidChange((e) => {
        if (
          e.key === ELEVENLABS_API_KEY_SECRET ||
          e.key === OPENAI_API_KEY_SECRET ||
          e.key === ANTHROPIC_API_KEY_SECRET
        ) {
          void (async () => {
            void wv.postMessage(
              await buildVocodePanelConfigMessage(this.extensionContext),
            );
          })();
        }
      }),
    );

    webviewView.onDidDispose(() => {
      for (const d of disposables) {
        d.dispose();
      }
    });
  }

  private postVoiceUiStatus(): void {
    if (!this.view) {
      return;
    }
    void this.view.webview.postMessage({
      type: "voiceUiStatus",
      state: this.services.voiceStatus.getState(),
    });
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

  private getHtml(webview: vscode.Webview, webviewRoot: vscode.Uri): string {
    const scriptUri = webview.asWebviewUri(
      vscode.Uri.joinPath(webviewRoot, "main-panel.js"),
    );
    const styleUri = webview.asWebviewUri(
      vscode.Uri.joinPath(webviewRoot, "main-panel.css"),
    );
    const csp = [
      "default-src 'none';",
      `style-src ${webview.cspSource} 'unsafe-inline';`,
      `script-src ${webview.cspSource};`,
    ].join(" ");

    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta http-equiv="Content-Security-Policy" content="${csp}" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <link href="${styleUri}" rel="stylesheet" />
</head>
<body>
  <div id="root"></div>
  <script type="module" src="${scriptUri}"></script>
</body>
</html>`;
  }

  dispose(): void {
    this.voiceStatusDisposable.dispose();
    this.unsubscribe();
  }
}

/** Pass to `registerWebviewViewProvider` — must match `package.json` `views` id. */
export const mainPanelViewType = mainPanelViewId;
