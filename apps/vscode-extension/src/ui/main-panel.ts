import * as vscode from "vscode";

import {
  applyVocodePanelConfigPatch,
  buildVocodePanelConfigMessage,
} from "../config/panel-settings";
import {
  ELEVENLABS_API_KEY_SECRET,
  elevenLabsApiKeyIsConfigured,
} from "../config/spawn-env";
import type { MainPanelStore } from "./main-panel-store";

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
  private readonly unsubscribe: () => void;

  constructor(
    private readonly extensionUri: vscode.Uri,
    private readonly store: MainPanelStore,
    private readonly extensionContext: vscode.ExtensionContext,
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

    const wv = webviewView.webview;
    const disposables: vscode.Disposable[] = [];

    disposables.push(
      wv.onDidReceiveMessage((msg: unknown) => {
        if (!msg || typeof msg !== "object") {
          return;
        }
        const m = msg as Record<string, unknown>;
        if (m.type === "webviewReady") {
          void (async () => {
            const ok = await elevenLabsApiKeyIsConfigured(
              this.extensionContext,
            );
            void wv.postMessage({
              type: "initialRoute",
              panelView: ok ? "main" : "settings",
            });
            void wv.postMessage(
              await buildVocodePanelConfigMessage(this.extensionContext),
            );
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
        if (m.type === "setElevenLabsApiKey") {
          const v = m.value;
          const s = typeof v === "string" ? v.trim() : "";
          void (async () => {
            try {
              if (s === "") {
                await this.extensionContext.secrets.delete(
                  ELEVENLABS_API_KEY_SECRET,
                );
              } else {
                await this.extensionContext.secrets.store(
                  ELEVENLABS_API_KEY_SECRET,
                  s,
                );
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
              await applyVocodePanelConfigPatch(
                patch as Record<string, unknown>,
              );
            } finally {
              void wv.postMessage(
                await buildVocodePanelConfigMessage(this.extensionContext),
              );
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
        if (e.key === ELEVENLABS_API_KEY_SECRET) {
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
    this.unsubscribe();
  }
}

/** Pass to `registerWebviewViewProvider` — must match `package.json` `views` id. */
export const mainPanelViewType = mainPanelViewId;
