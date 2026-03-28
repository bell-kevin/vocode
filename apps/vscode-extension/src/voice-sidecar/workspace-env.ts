import * as fs from "node:fs";
import * as path from "node:path";

/**
 * Merge workspace `.env` into `env` so spawned sidecar/daemon get the same vars as local CLI
 * (the extension host does not load `.env` into process.env).
 */
export function applyWorkspaceDotEnv(
  env: NodeJS.ProcessEnv,
  workspaceRoot: string | undefined,
): void {
  if (!workspaceRoot) {
    return;
  }
  const dotEnvPath = path.join(workspaceRoot, ".env");
  if (!fs.existsSync(dotEnvPath)) {
    return;
  }
  let content: string;
  try {
    content = fs.readFileSync(dotEnvPath, "utf8");
  } catch {
    return;
  }
  for (const line of content.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }
    const eq = trimmed.indexOf("=");
    if (eq <= 0) {
      continue;
    }
    const key = trimmed.slice(0, eq).trim();
    let val = trimmed.slice(eq + 1).trim();
    if (
      (val.startsWith('"') && val.endsWith('"')) ||
      (val.startsWith("'") && val.endsWith("'"))
    ) {
      val = val.slice(1, -1);
    }
    env[key] = val;
  }
}
