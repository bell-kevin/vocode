//animations from https://www.w3schools.com/w3css/default.asp

import { useId, useState } from "react"

/** Product copy — adjust URLs and details as the extension evolves */
const EXTENSION = {
    name: "Vocode",
    tagline: "Voice-driven AI code editing inside VS Code.",
    marketplaceUrl: "https://marketplace.visualstudio.com/items?itemName=publisher.extension-id",
    githubUrl: "https://github.com/your-org/your-extension-repo",
    docsUrl: "https://github.com/your-org/your-extension-repo#readme",
    marketplaceId: "publisher.extension-id",
}

const OVERVIEW = [
    "Vocode lets you speak code changes, and have them applied intelligently to your project using structured edits instead of raw text replacement.",
]

const BEBETTER = [
    "Speaking code changes allows thought about the code beyond explaining it in terms of code and more natural language.",
]

const WRITEFASTER = [
    "By speaking code changes, you can write faster without leaving the flow of your current task.",
]

const HOW_IT_WORKS = [
    {
        title: "VS Code Extension (TypeScript)",
        body: ["Captures voice + user intent", "Displays UI (transcripts, diffs, status)", "Sends requests to the daemon"],
    },
    {
        title: "Core Daemon (Go)",
        body: ["agent logic", "code edits (AST/diff-based)", "indexing (grep → symbols → AST)", "command execution", "transcript planning/orchestration"],
    },
    {
        title: "Voice Sidecar (Go)",
        body: ["native microphone capture", "STT provider integration", "transcript event emission to the extension"],
    },
]

/** Per-IDE install: step 1 is IDE-specific; 2–3 are shared behavior */
function installStepsForIde(ide) {
    const step1 = {
        vscode: (
            <>
                Open <strong>Extensions</strong> (<kbd>Ctrl+Shift+X</kbd> / <kbd>Cmd+Shift+X</kbd>) and install{" "}
                <strong>{EXTENSION.name}</strong>, or use the CLI below.
            </>
        ),
        vocodeide: (
            <>
                coming soon
            </>
        ),
        intellij: (
            <>
                coming soon
            </>
        ),
        neovim: (
            <>
                coming soon
            </>
        ),
    }
    return [
        step1[ide],
        <>
            Start or install the <strong>Vocode daemon</strong> per the docs so your editor can connect to Vocode.
        </>,
        <>Reload the window or restart the IDE if prompted, then open Vocode from the command palette and confirm the connection.</>,
    ]
}

const INSTALL_CLI = {
    vscode: { cmd: `code --install-extension ${EXTENSION.marketplaceId}`, label: "CLI" },
    cursor: { cmd: `cursor --install-extension ${EXTENSION.marketplaceId}`, label: "CLI" },
    intellij: null,
    neovim: { cmd: `neovim --install-extension ${EXTENSION.marketplaceId}`, label: "CLI (if available)" },
}

const INSTALL_TABS = [
    { id: "vscode", label: "VS Code" },
    { id: "vocodeide", label: "Vocode IDE" },
    { id: "intellij", label: "Intellij" },
    { id: "neovim", label: "Neovim" },
]

function renderRichText(text) {
    const parts = text.split(/(\*\*[^*]+\*\*)/g)
    return parts.map((part, i) => {
        const m = part.match(/^\*\*(.+)\*\*$/)
        if (m) return <strong key={i}>{m[1]}</strong>
        return part
    })
}

const btnBase =
    "inline-flex items-center justify-center rounded-md border border-transparent px-[1.15rem] py-[0.65rem] text-[0.95rem] font-semibold transition-colors"

const stepKbd =
    "[&_kbd]:rounded [&_kbd]:border [&_kbd]:border-black/15 [&_kbd]:bg-black/5 [&_kbd]:px-1.5 [&_kbd]:py-0.5 [&_kbd]:font-mono [&_kbd]:text-[0.85em] dark:[&_kbd]:border-white/20 dark:[&_kbd]:bg-white/8"

const tabBtn =
    "rounded-md border px-3 py-2 text-[0.9rem] font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500"

function HomePage() {
    const [installTab, setInstallTab] = useState("vscode")
    const installPanelId = useId()

    return (
        <>
        <header className="mb-5 py-12 bg-[url(https://images.unsplash.com/photo-1617994452722-4145e196248b?q=80&w=1170&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D)] bg-cover bg-center" aria-labelledby="hero-title">
            <h1
                id="hero-title"
                className="mb-3 text-[clamp(1.75rem,4vw,3.5rem)] font-normal leading-tight tracking-tight text-slate-900 dark:text-slate-100 w3-animate-opacity"
            >
                {EXTENSION.name}
            </h1>
            <p className="mb-4 text-xl font-semibold leading-snug text-neutral-900 dark:text-sky-200 w3-animate-opacity">
                {EXTENSION.tagline}
            </p>
            <div className="max-w-3xl w-screen mx-auto">
                <div className=" text-left text-[clamp(0.5rem,4vw,1.25rem)]">
                    {OVERVIEW.filter((p) => p?.trim()).map((p, i) => (
                        <p
                            key={i}
                            className="mb-4 leading-relaxed text-neutral-800 last:mb-0 dark:text-neutral-300"
                        >
                            {renderRichText(p)}
                        </p>
                    ))}
                </div>
            </div>
        </header>
        <div className="mx-auto max-w-6xl px-5 py-8 pb-12 text-left">
            

            

            <section className="mb-12 w3-animate-bottom" aria-labelledby="help-heading">
                <h2
                    id="help-heading"
                    className="mb-2 text-[1.4rem] font-normal tracking-tight text-slate-900 dark:font-semibold dark:text-sky-400"
                >
                    How it will let you be a better developer
                </h2>
                <div className="max-w-3xl">
                    {BEBETTER.filter((p) => p?.trim()).map((p, i) => (
                        <p
                            key={i}
                            className="mb-4 leading-relaxed text-neutral-800 last:mb-0 dark:text-neutral-300"
                        >
                            {renderRichText(p)}
                        </p>
                    ))}
                </div>
            </section>

            <section className="mb-12 w3-animate-bottom" aria-labelledby="faster-heading">
                <h2
                    id="faster-heading"
                    className="mb-2 text-[1.4rem] font-normal tracking-tight text-slate-900 dark:font-semibold dark:text-sky-400"
                >
                    Write Code Faster
                </h2>
                <div className="max-w-3xl">
                    {WRITEFASTER.filter((p) => p?.trim()).map((p, i) => (
                        <p
                            key={i}
                            className="mb-4 leading-relaxed text-neutral-800 last:mb-0 dark:text-neutral-300"
                        >
                            {renderRichText(p)}
                        </p>
                    ))}
                </div>
            </section>

            <section className="mb-12 w3-animate-right" aria-labelledby="how-heading">
                <h2
                    id="how-heading"
                    className="mb-2 text-[1.4rem] font-normal tracking-tight text-slate-900 dark:font-semibold dark:text-sky-400"
                >
                    How it works
                </h2>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                    {HOW_IT_WORKS.map((step, i) => (
                        <article
                            key={step.title}
                            className="flex items-start gap-4 rounded-[10px] border border-black/10 bg-white/65 p-4 dark:border-white/14 dark:bg-black/20 sm:p-[1.15rem]"
                        >
                            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[#007acc]/15 text-[0.95rem] font-bold text-sky-800 dark:bg-[rgba(100,181,255,0.14)] dark:text-sky-200">
                                {i + 1}
                            </span>
                            <div>
                                <h3 className="mb-1.5 text-[1.05rem] text-slate-900 dark:text-sky-100">{step.title}</h3>
                                {step.body.map((line, j) => (
                                    <p
                                        key={j}
                                        className="mb-2 text-[0.95rem] leading-snug text-neutral-600 last:mb-0 dark:text-neutral-400"
                                    >
                                        {renderRichText(line)}
                                    </p>
                                ))}
                            </div>
                        </article>
                    ))}
                </div>
            </section>

            <section className="mb-12" aria-labelledby="install-heading">
                <h2
                    id="install-heading"
                    className="mb-2 text-[1.4rem] font-normal tracking-tight text-slate-900 dark:font-semibold dark:text-sky-400"
                >
                    Install
                </h2>
                <p className="mb-4 max-w-3xl text-[0.95rem] text-neutral-600 dark:text-neutral-400">
                    Pick your editor. The core flow is the same: install the plugin, run the daemon, then connect from
                    the IDE.
                </p>
                <div
                    className="mb-4 flex flex-wrap gap-2 border-b border-black/10 pb-3 dark:border-white/15"
                    role="tablist"
                    aria-label="Choose IDE for install instructions"
                >
                    {INSTALL_TABS.map((t) => {
                        const selected = installTab === t.id
                        return (
                            <button
                                key={t.id}
                                type="button"
                                role="tab"
                                id={`install-tab-${t.id}`}
                                aria-selected={selected}
                                aria-controls={`${installPanelId}-${t.id}`}
                                tabIndex={selected ? 0 : -1}
                                className={`${tabBtn} ${
                                    selected
                                        ? "border-sky-600 bg-sky-50 text-sky-900 dark:border-sky-400/60 dark:bg-sky-950/50 dark:text-sky-100"
                                        : "border-black/12 bg-white/70 text-neutral-700 hover:border-sky-500/40 hover:bg-[#007acc]/8 dark:border-white/15 dark:bg-black/25 dark:text-neutral-300 dark:hover:bg-white/8"
                                }`}
                                onClick={() => setInstallTab(t.id)}
                            >
                                {t.label}
                            </button>
                        )
                    })}
                </div>
                {INSTALL_TABS.map((t) => {
                    const hidden = installTab !== t.id
                    const cli = INSTALL_CLI[t.id]
                    const steps = installStepsForIde(t.id)
                    return (
                        <div
                            key={t.id}
                            id={`${installPanelId}-${t.id}`}
                            role="tabpanel"
                            aria-labelledby={`install-tab-${t.id}`}
                            hidden={hidden}
                            className={hidden ? "hidden" : undefined}
                        >
                            <ol className="m-0 list-none p-0">
                                {steps.map((step, i) => (
                                    <li key={i} className="mb-4 flex items-start gap-4 leading-snug last:mb-0">
                                        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#007acc] text-[0.85rem] font-bold text-white">
                                            {i + 1}
                                        </span>
                                        <span
                                            className={`pt-0.5 text-neutral-800 dark:text-neutral-300 ${stepKbd}`}
                                        >
                                            {step}
                                        </span>
                                    </li>
                                ))}
                            </ol>
                            {cli ? (
                                <p className="mt-5 text-sm text-neutral-600 dark:text-neutral-400">
                                    {cli.label}:{" "}
                                    <code className="rounded bg-black/6 px-1.5 py-0.5 font-mono text-[0.82em] text-neutral-800 dark:bg-white/10 dark:text-neutral-200">
                                        {cli.cmd}
                                    </code>
                                </p>
                            ) : (
                                <p className="mt-5 text-sm text-neutral-600 dark:text-neutral-400">
                                    coming soon
                                </p>
                            )}
                        </div>
                    )
                })}
                <div className="mt-5 flex flex-wrap gap-3">
                    <a
                        className={`${btnBase} bg-[#007acc] text-white shadow-sm hover:bg-[#0062a3]`}
                        href={EXTENSION.marketplaceUrl}
                        rel="noopener noreferrer"
                    >
                        Marketplace listing
                    </a>
                    <a
                        className={`${btnBase} border-sky-600/45 bg-white/85 text-sky-800 hover:border-sky-600 hover:bg-white hover:text-sky-900 dark:border-sky-400/40 dark:bg-white/10 dark:text-sky-300 dark:hover:bg-white/15`}
                        href={EXTENSION.githubUrl}
                        rel="noopener noreferrer"
                    >
                        Source on GitHub
                    </a>
                </div>
            </section>

            <section
                className="rounded-xl bg-gradient-to-br from-[#1e1e1e] via-[#252526] to-[#1a2f44] p-8 text-center text-neutral-200"
                aria-labelledby="bottom-cta"
            >
                <h2
                    id="bottom-cta"
                    className="mb-2 text-[1.35rem] font-semibold text-slate-100 dark:text-sky-200"
                >
                    Learn more
                </h2>
                <p className="mx-auto mb-5 max-w-lg text-[0.95rem] leading-relaxed text-neutral-200/90">
                    Read setup for the daemon, privacy notes, and troubleshooting in the project documentation.
                </p>
                <div className="flex flex-wrap justify-center gap-3">
                    <a
                        className={`${btnBase} bg-[#007acc] text-white hover:bg-[#1c97ea]`}
                        href={EXTENSION.docsUrl}
                        rel="noopener noreferrer"
                    >
                        Documentation
                    </a>
                    <a
                        className={`${btnBase} border border-white/35 bg-transparent text-neutral-200 hover:border-white/55 hover:bg-white/8`}
                        href={EXTENSION.githubUrl}
                        rel="noopener noreferrer"
                    >
                        GitHub
                    </a>
                </div>
            </section>
        </div>
        </>
    )
}

export default HomePage
