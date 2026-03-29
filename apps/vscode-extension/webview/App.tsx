import { useEffect, useRef, useState } from "react";

import { AudioMeter } from "./AudioMeter";
import { MainPanel } from "./MainPanel";
import { vocodePanelConfigFromMessage } from "./panel-config-from-message";
import type { VocodePanelConfig } from "./panel-config-types";
import { SettingsPanel } from "./SettingsPanel";
import type { PanelState } from "./types";
import { emptyState, normalizePanelState } from "./util";
import { getVsCodeApi } from "./vscode-api";

function GearIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  );
}

function ChevronLeftIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M15 18l-6-6 6-6" />
    </svg>
  );
}

export function App() {
  const [panel, setPanel] = useState<PanelState>(emptyState);
  const [panelView, setPanelView] = useState<"main" | "settings">("main");
  const [panelConfig, setPanelConfig] = useState<VocodePanelConfig | null>(
    null,
  );
  const initialRouteApplied = useRef(false);

  useEffect(() => {
    const handler = (event: MessageEvent) => {
      const msg = event.data as Record<string, unknown> | null;
      if (!msg || typeof msg !== "object") {
        return;
      }
      if (msg.type === "update" && msg.state !== undefined) {
        setPanel(normalizePanelState(msg.state));
      }
      if (msg.type === "initialRoute" && !initialRouteApplied.current) {
        initialRouteApplied.current = true;
        const v = msg.panelView;
        if (v === "settings" || v === "main") {
          setPanelView(v);
        }
      }
      if (msg.type === "panelConfig") {
        setPanelConfig(vocodePanelConfigFromMessage(msg));
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, []);

  useEffect(() => {
    getVsCodeApi()?.postMessage({ type: "webviewReady" });
  }, []);

  useEffect(() => {
    if (panelView === "settings") {
      getVsCodeApi()?.postMessage({ type: "requestPanelConfig" });
    }
  }, [panelView]);

  return (
    <div className="app-shell">
      <header className="panel-top">
        <div className="panel-top-title">
          {panelView === "main" ? "Voice" : "Settings"}
        </div>
        <div
          className="panel-top-actions"
          role="toolbar"
          aria-label="Panel actions"
        >
          {panelView === "main" ? (
            <button
              type="button"
              className="panel-icon-btn"
              aria-label="Settings"
              title="Settings"
              onClick={() => setPanelView("settings")}
            >
              <GearIcon />
            </button>
          ) : (
            <button
              type="button"
              className="panel-icon-btn"
              aria-label="Back to Voice"
              title="Back"
              onClick={() => setPanelView("main")}
            >
              <ChevronLeftIcon />
            </button>
          )}
        </div>
      </header>
      {panelView === "main" ? (
        <>
          <AudioMeter state={panel} />
          <MainPanel state={panel} />
        </>
      ) : (
        <SettingsPanel config={panelConfig} />
      )}
    </div>
  );
}
