import { useLayoutEffect, useRef } from "react";

import type { PanelState } from "./types";

function drawWaveform(
  canvas: HTMLCanvasElement | null,
  samples: readonly number[],
  opts?: { listening?: boolean; rms?: number },
) {
  if (!canvas?.getContext) {
    return;
  }
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    return;
  }
  const w = canvas.width;
  const h = canvas.height;
  ctx.clearRect(0, 0, w, h);
  const fg =
    getComputedStyle(document.body)
      .getPropertyValue("--vscode-textLink-foreground")
      .trim() || "#3794ff";
  const arr = Array.isArray(samples) ? samples : [];
  if (arr.length === 0) {
    if (opts?.listening === true) {
      const level = Math.min(1, Math.max(0, opts.rms ?? 0));
      const pad = 4;
      const innerW = w - pad * 2;
      const barH = Math.max(6, Math.min(h - pad * 2, 14));
      const y = h - pad - barH;
      ctx.fillStyle = fg;
      ctx.globalAlpha = 0.2;
      ctx.fillRect(pad, y, innerW, barH);
      ctx.globalAlpha = 0.55 + 0.35 * level;
      const fillW = Math.max(level > 0 ? 3 : 0, innerW * level);
      ctx.fillRect(pad, y, fillW, barH);
      ctx.globalAlpha = 1;
      return;
    }
    ctx.strokeStyle = fg;
    ctx.globalAlpha = 0.22;
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(0, h - 2);
    ctx.lineTo(w, h - 2);
    ctx.stroke();
    ctx.setLineDash([]);
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

export function AudioMeter(props: { state: PanelState }) {
  const { state } = props;
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const voiceListening = state.voiceListening === true;
  const am = state.audioMeter;
  const rms = typeof am.rms === "number" ? am.rms : 0;
  const speaking = am.speaking === true;
  const pct = Math.round(Math.min(1, Math.max(0, rms)) * 100);

  useLayoutEffect(() => {
    drawWaveform(canvasRef.current, am.waveform ?? [], {
      listening: voiceListening,
      rms,
    });
  }, [am.waveform, voiceListening, rms]);

  return (
    <div className="meter card">
      <div className="meta">
        <span className="badge">
          {!voiceListening ? "Idle" : speaking ? "Speaking" : "Quiet"}
        </span>
        <span>{!voiceListening ? "Not listening" : "Input level"}</span>
      </div>
      <div className="meter-bar">
        <div
          className="meter-fill"
          style={{
            width: voiceListening
              ? pct <= 0
                ? "4px"
                : `max(${pct}%, 4px)`
              : `${pct}%`,
          }}
        />
      </div>
      <canvas
        ref={canvasRef}
        className="wave-canvas"
        width={320}
        height={44}
        aria-label="Recent level"
      />
    </div>
  );
}
