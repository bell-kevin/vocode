import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useEffect, useRef, useState } from "react";

import { getVsCodeApi } from "./api/vscode";
import type { VocodeConfig } from "./config";
import { vocodeConfigFromMessage } from "./config";
import { ClarifyPanel, MainPanel, SearchPanel, SettingsPanel } from "./panels";
import { ProcessingStrip } from "./panels/processing-strip";
import type { PanelState } from "./types";
import { emptyState, normalizePanelState } from "./util";
import { type VoiceUiStatus, VoiceVisualization } from "./voice-visualization";

type PanelView = "main" | "settings" | "clarify" | "search";

function handleHostPanelMessage(
  msg: Record<string, unknown>,
  initialRouteApplied: MutableRefObject<boolean>,
  setPanel: Dispatch<SetStateAction<PanelState>>,
  setPanelView: Dispatch<SetStateAction<PanelView>>,
  setPanelConfig: Dispatch<SetStateAction<VocodeConfig | null>>,
  setVoiceUiStatus: Dispatch<SetStateAction<VoiceUiStatus>>,
): void {
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
    setPanelConfig(vocodeConfigFromMessage(msg));
  }
  if (msg.type === "openPanelView") {
    const v = msg.panelView;
    if (v === "settings" || v === "main") {
      setPanelView(v);
    }
  }
  if (msg.type === "voiceUiStatus") {
    const s = msg.state;
    if (s === "idle" || s === "listening" || s === "processing") {
      setVoiceUiStatus(s);
    }
  }
}

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

function panelTitle(view: PanelView): string {
  switch (view) {
    case "settings":
      return "Settings";
    case "clarify":
      return "Clarification";
    case "search":
      return "Search";
    default:
      return "Vocode";
  }
}

export function App() {
  const [panel, setPanel] = useState<PanelState>(emptyState);
  const [panelView, setPanelView] = useState<PanelView>("main");
  const [panelConfig, setPanelConfig] = useState<VocodeConfig | null>(null);
  const [voiceUiStatus, setVoiceUiStatus] = useState<VoiceUiStatus>("idle");
  const initialRouteApplied = useRef(false);

  const hasClarifyInterrupt = Boolean(panel.clarifyPrompt?.question);
  const hasSearchInterrupt =
    Array.isArray(panel.searchState?.results) &&
    panel.searchState.results.length > 0;

  useEffect(() => {
    const handler = (event: MessageEvent) => {
      const msg = event.data as Record<string, unknown> | null;
      if (!msg || typeof msg !== "object") {
        return;
      }
      handleHostPanelMessage(
        msg,
        initialRouteApplied,
        setPanel,
        setPanelView,
        setPanelConfig,
        setVoiceUiStatus,
      );
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

  useEffect(() => {
    setPanelView((prev) => {
      if (prev === "settings") {
        return prev;
      }
      if (hasClarifyInterrupt) {
        return "clarify";
      }
      if (hasSearchInterrupt) {
        return "search";
      }
      if (prev === "clarify" || prev === "search") {
        return "main";
      }
      return prev;
    });
  }, [hasClarifyInterrupt, hasSearchInterrupt]);

  const handleHeaderBack = () => {
    if (panelView === "settings") {
      setPanelView("main");
    }
  };

  const showGear = panelView === "main";
  /** Clarify / search are not exited from the header — use in-panel actions. */
  const showBack = panelView === "settings";

  return (
    <div className="app-shell">
      <header className="panel-top">
        <div className="panel-top-title">{panelTitle(panelView)}</div>
        <div
          className="panel-top-actions"
          role="toolbar"
          aria-label="Panel actions"
        >
          {showGear ? (
            <button
              type="button"
              className="panel-icon-btn"
              aria-label="Settings"
              title="Settings"
              onClick={() => setPanelView("settings")}
            >
              <GearIcon />
            </button>
          ) : null}
          {showBack ? (
            <button
              type="button"
              className="panel-icon-btn"
              aria-label="Back to Vocode"
              title="Back"
              onClick={handleHeaderBack}
            >
              <ChevronLeftIcon />
            </button>
          ) : null}
        </div>
      </header>
      <div className="app-body">
        {panelView === "main" ? (
          <>
            <div className="voice-viz-slot">
              <VoiceVisualization state={panel} voiceUiStatus={voiceUiStatus} />
            </div>
            <MainPanel state={panel} />
          </>
        ) : null}
        {panelView === "settings" ? (
          <SettingsPanel config={panelConfig} />
        ) : null}
        {panelView === "clarify" ? (
          <>
            <div className="voice-viz-slot">
              <VoiceVisualization state={panel} voiceUiStatus={voiceUiStatus} />
            </div>
            <ProcessingStrip pending={panel.pending} />
            <ClarifyPanel state={panel} />
          </>
        ) : null}
        {panelView === "search" ? (
          <>
            <div className="voice-viz-slot">
              <VoiceVisualization state={panel} voiceUiStatus={voiceUiStatus} />
            </div>
            <ProcessingStrip pending={panel.pending} />
            <SearchPanel state={panel} />
          </>
        ) : null}
      </div>
    </div>
  );
}
