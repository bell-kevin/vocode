/** Marketing + install metadata — keep in sync with real extension publication when you ship */
export const SITE = {
  name: "Vocode",
  /** Short line under logo in nav / meta */
  shortTagline: "Voice-first agentic programming",
  marketplaceUrl:
    "https://marketplace.visualstudio.com/items?itemName=publisher.extension-id",
  githubUrl: "https://github.com/your-org/your-extension-repo",
  docsUrl: "https://github.com/your-org/your-extension-repo#readme",
  marketplaceId: "publisher.extension-id",
};

export const INSTALL_CLI = {
  vscode: {
    cmd: `code --install-extension ${SITE.marketplaceId}`,
    label: "CLI",
  },
  cursor: {
    cmd: `cursor --install-extension ${SITE.marketplaceId}`,
    label: "CLI",
  },
  intellij: null,
  neovim: {
    cmd: `neovim --install-extension ${SITE.marketplaceId}`,
    label: "CLI (if available)",
  },
};

export const INSTALL_TABS = [
  { id: "vscode", label: "VS Code" },
  { id: "vocodeide", label: "Vocode IDE" },
  { id: "intellij", label: "IntelliJ" },
  { id: "neovim", label: "Neovim" },
];
