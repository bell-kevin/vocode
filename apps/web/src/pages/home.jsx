import { useId, useState } from "react";

import { INSTALL_CLI, INSTALL_TABS, SITE } from "../site.js";

const PILLARS = [
  {
    title: "Speak, don’t re-type",
    body: "Dictate refactors, edits, and navigation without breaking flow. Vocode turns what you say into structured actions—not a wall of model text to paste and pray over.",
  },
  {
    title: "Your editor, amplified",
    body: "Stay in your editor. Keep your shortcuts, themes, and muscle memory. Voice becomes another input—fast when you want it, silent when you don’t.",
  },
  {
    title: "Built to ship",
    body: "Safety and validation stay on the engine side; the editor stays deterministic. You get a repeatable loop: speak → review → apply—with room to grow into how your team works.",
  },
];

const FEATURES = [
  {
    title: "Structured edits",
    description:
      "Changes map to real edit and command directives—so results align with your tree, your files, and your expectations.",
    accent: "from-[#4f81ff]/25 to-transparent",
  },
  {
    title: "Voice-native loop",
    description:
      "Microphone capture and transcription are first-class—tuned for short, actionable utterances while you code.",
    accent: "from-cyan-500/15 to-transparent",
  },
  {
    title: "Iterate in place",
    description:
      "Transcripts, batches, and outcomes stay beside your work so you can course-correct in seconds—not re-prompt from scratch.",
    accent: "from-violet-500/15 to-transparent",
  },
];

/** Per-IDE install: step 1 is IDE-specific; 2–3 are shared behavior */
function installStepsForIde(ide) {
  const step1 = {
    vscode: (
      <>
        Open <strong>Extensions</strong> (<kbd>Ctrl+Shift+X</kbd> /{" "}
        <kbd>Cmd+Shift+X</kbd>) and install <strong>{SITE.name}</strong>, or use
        the CLI below.
      </>
    ),
    vocodeide: <>coming soon</>,
    intellij: <>coming soon</>,
    neovim: <>coming soon</>,
  };
  return [
    {
      id: `${ide}-step-1`,
      content: step1[ide],
    },
    {
      id: `${ide}-step-2`,
      content: (
        <>
          Start or install the <strong>Vocode daemon</strong> per the docs so
          your editor can connect.
        </>
      ),
    },
    {
      id: `${ide}-step-3`,
      content: (
        <>
          Reload the window if prompted, open the Vocode panel, and confirm
          you’re connected.
        </>
      ),
    },
  ];
}

function renderRichText(text) {
  let seq = 0;
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((part) => {
    const key = `rt-${seq++}`;
    const m = part.match(/^\*\*(.+)\*\*$/);
    if (m) {
      return <strong key={key}>{m[1]}</strong>;
    }
    return <span key={key}>{part}</span>;
  });
}

const btnPrimary =
  "inline-flex items-center justify-center rounded-md border border-transparent bg-[#4f81ff] px-6 py-3 text-[0.95rem] font-semibold text-white shadow-[0_0_32px_-10px_rgba(79,129,255,0.9)] transition-colors hover:bg-[#3d6fe6]";

const btnGhost =
  "inline-flex items-center justify-center rounded-md border border-white/20 bg-white/5 px-6 py-3 text-[0.95rem] font-semibold text-white transition-colors hover:border-white/35 hover:bg-white/10";

const stepKbd =
  "[&_kbd]:rounded [&_kbd]:border [&_kbd]:border-white/20 [&_kbd]:bg-white/8 [&_kbd]:px-1.5 [&_kbd]:py-0.5 [&_kbd]:font-mono [&_kbd]:text-[0.85em]";

const tabBtn =
  "rounded-lg border px-3 py-2 text-[0.9rem] font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#4f81ff]";

function HomePage() {
  const [installTab, setInstallTab] = useState("vscode");
  const installPanelId = useId();

  return (
    <>
      <section
        className="relative overflow-hidden border-b border-white/[0.06]"
        aria-labelledby="hero-title"
      >
        <div className="pointer-events-none absolute inset-0" aria-hidden>
          <div className="absolute -top-[55%] left-1/2 h-[min(900px,120vw)] w-[min(900px,120vw)] -translate-x-1/2 rounded-full bg-[#4f81ff]/18 blur-[100px]" />
          <div className="absolute -bottom-32 right-[-10%] h-[420px] w-[420px] rounded-full bg-cyan-500/12 blur-[90px]" />
          <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(6,6,6,0)_0%,#060606_88%)]" />
        </div>

        <div className="relative mx-auto max-w-6xl px-5 pb-24 pt-20 sm:pb-28 sm:pt-24 lg:pt-28">
          <p className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-xs font-medium uppercase tracking-[0.14em] text-neutral-400">
            <span className="h-1.5 w-1.5 rounded-full bg-[#4f81ff] shadow-[0_0_8px_#4f81ff]" />
            New programming surface
          </p>
          <h1
            id="hero-title"
            className="max-w-4xl text-left text-[clamp(2.25rem,6vw,3.75rem)] font-semibold leading-[1.08] tracking-tight text-white"
          >
            Code at the speed
            <br />
            <span className="bg-gradient-to-r from-white via-[#7396ff] to-[#4f81ff] bg-clip-text text-transparent">
              you speak
            </span>
            .
          </h1>
          <p className="mt-6 max-w-2xl text-left text-lg leading-relaxed text-neutral-400 sm:text-xl">
            Vocode is voice-first agentic programming: speak intent, get
            structured changes in{" "}
            <strong className="font-medium text-neutral-200">
              your favorite editor
            </strong>
            , and never snap out of the loop. This isn’t a demo repo—it’s the
            paradigm you’ll wish you had years ago.
          </p>
          <div className="mt-10 flex flex-wrap items-center gap-4">
            <a
              className={btnPrimary}
              href={SITE.marketplaceUrl}
              rel="noopener noreferrer"
            >
              Start with VS Code
            </a>
            <a className={btnGhost} href={SITE.docsUrl}>
              Read the docs
            </a>
          </div>
          <p className="mt-6 text-left text-sm text-neutral-500">
            No account. Install the extension, run the engine locally, and
            you’re in.
          </p>
        </div>
      </section>

      <div className="mx-auto max-w-6xl px-5 py-20 sm:py-24">
        <section className="mb-20 text-left" aria-labelledby="pillars-heading">
          <h2
            id="pillars-heading"
            className="mb-3 text-sm font-semibold uppercase tracking-[0.16em] text-[#4f81ff]"
          >
            Why teams will switch
          </h2>
          <p className="mb-12 max-w-2xl text-2xl font-medium leading-snug text-white sm:text-3xl">
            {renderRichText(
              "The editor isn’t going away—how you drive it is. **Speech** becomes precision tooling, not a party trick.",
            )}
          </p>
          <ul className="grid gap-6 sm:grid-cols-3">
            {PILLARS.map((item) => (
              <li
                key={item.title}
                className="rounded-xl border border-white/[0.08] bg-white/[0.02] p-6"
              >
                <h3 className="mb-3 text-base font-semibold text-white">
                  {item.title}
                </h3>
                <p className="text-[0.95rem] leading-relaxed text-neutral-400">
                  {renderRichText(item.body)}
                </p>
              </li>
            ))}
          </ul>
        </section>

        <section className="mb-20 text-left" aria-labelledby="features-heading">
          <h2
            id="features-heading"
            className="mb-10 text-sm font-semibold uppercase tracking-[0.16em] text-neutral-500"
          >
            Product surface
          </h2>
          <div className="grid gap-5 md:grid-cols-3">
            {FEATURES.map((f) => (
              <article
                key={f.title}
                className={`relative overflow-hidden rounded-2xl border border-white/[0.08] bg-gradient-to-br ${f.accent} p-6`}
              >
                <h3 className="relative text-lg font-semibold text-white">
                  {f.title}
                </h3>
                <p className="relative mt-2 text-[0.95rem] leading-relaxed text-neutral-400">
                  {f.description}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="mb-20" aria-labelledby="install-heading">
          <h2
            id="install-heading"
            className="mb-2 text-left text-2xl font-semibold tracking-tight text-white"
          >
            Install in minutes
          </h2>
          <p className="mb-8 max-w-2xl text-left text-neutral-400">
            {renderRichText(
              "Plug into your existing stack. Pick an editor—the core loop is the same: **extension + engine + microphone**.",
            )}
          </p>
          <div
            className="mb-6 flex flex-wrap gap-2 border-b border-white/10 pb-4"
            role="tablist"
            aria-label="Choose editor for install steps"
          >
            {INSTALL_TABS.map((t) => {
              const selected = installTab === t.id;
              return (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  id={`home-install-tab-${t.id}`}
                  aria-selected={selected}
                  aria-controls={`${installPanelId}-${t.id}`}
                  tabIndex={selected ? 0 : -1}
                  className={`${tabBtn} ${
                    selected
                      ? "border-[#4f81ff]/50 bg-[#4f81ff]/12 text-white"
                      : "border-white/12 bg-black/30 text-neutral-400 hover:border-[#4f81ff]/35 hover:bg-white/5"
                  }`}
                  onClick={() => setInstallTab(t.id)}
                >
                  {t.label}
                </button>
              );
            })}
          </div>
          {INSTALL_TABS.map((t) => {
            const hidden = installTab !== t.id;
            const cli = INSTALL_CLI[t.id];
            const steps = installStepsForIde(t.id);
            return (
              <div
                key={t.id}
                id={`${installPanelId}-${t.id}`}
                role="tabpanel"
                aria-labelledby={`home-install-tab-${t.id}`}
                hidden={hidden}
                className={hidden ? "hidden" : undefined}
              >
                <ol className="m-0 list-none p-0 text-left">
                  {steps.map((step, i) => (
                    <li
                      key={step.id}
                      className="mb-5 flex items-start gap-4 leading-snug last:mb-0"
                    >
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#4f81ff] text-[0.8rem] font-bold text-white">
                        {i + 1}
                      </span>
                      <span className={`pt-1 text-neutral-300 ${stepKbd}`}>
                        {step.content}
                      </span>
                    </li>
                  ))}
                </ol>
                {cli ? (
                  <p className="mt-4 text-sm text-neutral-500">
                    {cli.label}:{" "}
                    <code className="rounded-md bg-white/10 px-2 py-1 font-mono text-[0.82em] text-neutral-200">
                      {cli.cmd}
                    </code>
                  </p>
                ) : (
                  <p className="mt-4 text-sm text-neutral-500">coming soon</p>
                )}
              </div>
            );
          })}
          <div className="mt-8 flex flex-wrap gap-3">
            <a
              className={`${btnPrimary} px-5 py-2.5 text-sm`}
              href={SITE.marketplaceUrl}
              rel="noopener noreferrer"
            >
              Marketplace
            </a>
            <a
              className={`${btnGhost} px-5 py-2.5 text-sm`}
              href={SITE.githubUrl}
              rel="noopener noreferrer"
            >
              Source
            </a>
          </div>
        </section>

        <section
          className="relative overflow-hidden rounded-2xl border border-white/[0.08] bg-gradient-to-br from-[#4f81ff]/20 via-[#0a0f1a] to-[#060606] px-8 py-14 text-center sm:px-12"
          aria-labelledby="cta-heading"
        >
          <div
            className="pointer-events-none absolute inset-0 opacity-60"
            aria-hidden
          >
            <div className="absolute left-1/2 top-0 h-px w-3/4 max-w-lg -translate-x-1/2 bg-gradient-to-r from-transparent via-white/30 to-transparent" />
          </div>
          <h2
            id="cta-heading"
            className="relative text-2xl font-semibold tracking-tight text-white sm:text-3xl"
          >
            Try the way you’ll build next.
          </h2>
          <p className="relative mx-auto mt-4 max-w-xl text-neutral-300">
            {renderRichText(
              "If you’re tired of tab-dancing between chat and code, Vocode is the shortcut back to **shipping**.",
            )}
          </p>
          <div className="relative mt-8 flex flex-wrap justify-center gap-4">
            <a className={btnPrimary} href={SITE.marketplaceUrl}>
              Get the extension
            </a>
            <a className={btnGhost} href={SITE.githubUrl}>
              Star on GitHub
            </a>
          </div>
        </section>
      </div>
    </>
  );
}

export default HomePage;
