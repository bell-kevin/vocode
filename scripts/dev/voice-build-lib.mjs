/**
 * Shared PortAudio (CGO) builds for vocode-voiced.
 * - Native: Windows (MSYS2), macOS (Homebrew portaudio), Linux (distro packages).
 * - Docker: Debian bookworm (glibc) for linux-*; Alpine (musl) for alpine-*.
 */

import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));

/** GitHub windows-11-arm can misreport process.arch; job id is reliable. */
const forceWinArm64VoiceBuild =
  process.env.GITHUB_JOB === "voice-windows-arm64" ||
  process.env.VOCODE_VOICE_FORCE_WIN_ARM64 === "1";

function msysVoiceSubdir() {
  if (forceWinArm64VoiceBuild) {
    return "clangarm64";
  }
  if (process.platform === "win32" && process.arch === "arm64") {
    return "clangarm64";
  }
  return "mingw64";
}

/** @returns {string} */
export function defaultRepoRoot() {
  return path.join(scriptDir, "..", "..");
}

/** @returns {{ slug: string; goos: string; goarch: string; goarm?: string }[]} */
export function loadFatTargets(repoRoot = defaultRepoRoot()) {
  const p = path.join(repoRoot, "scripts", "dev", "vsix-fat-targets.json");
  return JSON.parse(readFileSync(p, "utf8"));
}

/** @returns {string} e.g. "1.26" */
export function readGoVersionFromMod(repoRoot = defaultRepoRoot()) {
  const mod = readFileSync(path.join(repoRoot, "go.mod"), "utf8");
  const m = /^go\s+(\d+\.\d+)/m.exec(mod);
  if (!m) {
    throw new Error("Could not parse go version from go.mod");
  }
  return m[1];
}

/** VS Code marketplace folder for this Node process. */
export function getHostSlug() {
  const p = process.platform;
  const a = process.arch;
  if (p === "win32") {
    return a === "arm64" ? "win32-arm64" : "win32-x64";
  }
  if (p === "darwin") {
    return a === "arm64" ? "darwin-arm64" : "darwin-x64";
  }
  if (p === "linux") {
    if (a === "arm64") return "linux-arm64";
    if (a === "arm") return "linux-armhf";
    return "linux-x64";
  }
  throw new Error(`Unsupported host: ${p} ${a}`);
}

/**
 * @param {string} hostPath
 * @returns {string} Docker bind source path (Docker Desktop on Windows uses //c/... )
 */
export function hostPathForDockerBind(hostPath) {
  const resolved = path.resolve(hostPath);
  if (process.platform !== "win32") {
    return resolved;
  }
  const norm = resolved.replace(/\\/g, "/");
  const m = norm.match(/^([A-Za-z]):\/(.*)$/);
  if (m) {
    return `//${m[1].toLowerCase()}/${m[2]}`;
  }
  return norm;
}

export function dockerAvailable() {
  const r = spawnSync("docker", ["info"], {
    stdio: "ignore",
    shell: process.platform === "win32",
  });
  return r.status === 0;
}

/** @returns {{ ok: boolean; env: Record<string, string>; msysRoot?: string }} */
export function configurePortAudioMsys2() {
  const msysRoot = process.env.MSYS2_ROOT ?? "C:\\tools\\msys64";
  const subdir = msysVoiceSubdir();
  const mingwBin = path.join(msysRoot, subdir, "bin");
  const mingwPkgConfigDir = path.join(msysRoot, subdir, "lib", "pkgconfig");
  const pkgConfigExe = path.join(mingwBin, "pkg-config.exe");
  const pkgConfExe = path.join(mingwBin, "pkgconf.exe");
  const pkgTool = existsSync(pkgConfigExe)
    ? pkgConfigExe
    : existsSync(pkgConfExe)
      ? pkgConfExe
      : "";
  const gccExe = path.join(mingwBin, "gcc.exe");
  const clangExe = path.join(mingwBin, "clang.exe");
  const ccExe = existsSync(gccExe) ? gccExe : clangExe;

  const ok =
    pkgTool !== "" && existsSync(mingwPkgConfigDir) && existsSync(ccExe);
  if (!ok) {
    return { ok: false, env: {}, msysRoot };
  }

  const pathSep = ";";
  const pathPrefix = mingwBin + pathSep;
  const existingPath = process.env.Path ?? process.env.PATH ?? "";

  return {
    ok: true,
    env: {
      PATH: pathPrefix + existingPath,
      Path: pathPrefix + existingPath,
      PKG_CONFIG_PATH:
        mingwPkgConfigDir +
        (process.env.PKG_CONFIG_PATH
          ? pathSep + process.env.PKG_CONFIG_PATH
          : ""),
      PKG_CONFIG: pkgTool,
      CC: ccExe,
    },
  };
}

/** @returns {Record<string, string>} */
export function configurePortAudioDarwin() {
  const pr = spawnSync("brew", ["--prefix", "portaudio"], {
    encoding: "utf8",
  });
  if (pr.status !== 0 || !pr.stdout?.trim()) {
    throw new Error(
      "Homebrew portaudio not found. Install: brew install portaudio",
    );
  }
  const prefix = pr.stdout.trim();
  const pcDir = path.join(prefix, "lib", "pkgconfig");
  return {
    PKG_CONFIG_PATH:
      pcDir +
      (process.env.PKG_CONFIG_PATH ? `:${process.env.PKG_CONFIG_PATH}` : ""),
  };
}

/**
 * @param {string} slug
 * @param {string} repoRoot
 */
function targetForSlug(slug, repoRoot) {
  const t = loadFatTargets(repoRoot).find((x) => x.slug === slug);
  if (!t) {
    throw new Error(`Unknown VSIX slug: ${slug}`);
  }
  return t;
}

const debianPlatform = {
  "linux-x64": "linux/amd64",
  "linux-arm64": "linux/arm64",
  "linux-armhf": "linux/arm/v7",
};

/**
 * @param {string} slug
 * @param {string} repoRoot
 */
export function buildVoiceLinuxDockerDebian(slug, repoRoot) {
  const platform = debianPlatform[slug];
  if (!platform) {
    throw new Error(`Not a Debian-docker linux slug: ${slug}`);
  }
  if (!dockerAvailable()) {
    throw new Error(
      `Docker is required to build ${slug} on this machine. Install Docker Desktop or run the Linux voice CI job.`,
    );
  }

  const goVer = readGoVersionFromMod(repoRoot);
  const image = `golang:${goVer}-bookworm`;
  const t = targetForSlug(slug, repoRoot);
  const outHost = path.join(repoRoot, "apps", "voice", "bin", slug);
  mkdirSync(outHost, { recursive: true });

  const srcBind = hostPathForDockerBind(repoRoot);
  const outBind = hostPathForDockerBind(outHost);

  const goarm = t.goarm ? `export GOARM=${t.goarm}\n` : "";
  // Avoid bash -l: login shells reset PATH and drop /usr/local/go/bin from the golang image.
  const inner = `set -euo pipefail
export PATH="/usr/local/go/bin:$PATH"
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq pkg-config libportaudio2 portaudio19-dev >/dev/null
cd /src
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=${t.goarch}
${goarm}go build -buildvcs=false -trimpath -o /out/vocode-voiced ./apps/voice/cmd/vocode-voiced
`;

  const args = [
    "run",
    "--rm",
    "--platform",
    platform,
    "-v",
    `${srcBind}:/src:ro`,
    "-v",
    `${outBind}:/out`,
    image,
    "bash",
    "-c",
    inner,
  ];

  console.log(`[voice-build] ${slug}: docker ${image} (${platform})`);
  const r = spawnSync("docker", args, {
    stdio: "inherit",
    shell: process.platform === "win32",
  });
  if (r.error) {
    throw r.error;
  }
  if (r.status !== 0) {
    throw new Error(`docker build failed for ${slug} (exit ${r.status})`);
  }
}

const alpinePlatform = {
  "alpine-x64": "linux/amd64",
  "alpine-arm64": "linux/arm64",
};

/**
 * @param {string} slug
 * @param {string} repoRoot
 */
export function buildVoiceLinuxDockerAlpine(slug, repoRoot) {
  const platform = alpinePlatform[slug];
  if (!platform) {
    throw new Error(`Not an Alpine-docker slug: ${slug}`);
  }
  if (!dockerAvailable()) {
    throw new Error(
      `Docker is required to build ${slug}. Install Docker Desktop or use CI.`,
    );
  }

  const goVer = readGoVersionFromMod(repoRoot);
  const image = `golang:${goVer}-alpine`;
  const t = targetForSlug(slug, repoRoot);
  const outHost = path.join(repoRoot, "apps", "voice", "bin", slug);
  mkdirSync(outHost, { recursive: true });

  const srcBind = hostPathForDockerBind(repoRoot);
  const outBind = hostPathForDockerBind(outHost);

  const inner = `set -eu
export PATH="/usr/local/go/bin:$PATH"
apk add --no-cache --quiet pkgconfig build-base portaudio-dev
cd /src
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=${t.goarch}
go build -buildvcs=false -trimpath -o /out/vocode-voiced ./apps/voice/cmd/vocode-voiced
`;

  console.log(`[voice-build] ${slug}: docker ${image} (${platform})`);
  const r = spawnSync(
    "docker",
    [
      "run",
      "--rm",
      "--platform",
      platform,
      "-v",
      `${srcBind}:/src:ro`,
      "-v",
      `${outBind}:/out`,
      image,
      "sh",
      "-c",
      inner,
    ],
    { stdio: "inherit", shell: process.platform === "win32" },
  );
  if (r.error) {
    throw r.error;
  }
  if (r.status !== 0) {
    throw new Error(
      `docker alpine build failed for ${slug} (exit ${r.status})`,
    );
  }
}

/**
 * Native `go build` from module root with CGO.
 * @param {object} opts
 * @param {string} opts.repoRoot
 * @param {string} opts.slug - output folder under apps/voice/bin
 * @param {string} opts.outFile - final binary path
 * @param {Record<string, string>} [opts.extraEnv]
 */
export function goBuildVoiceNative(opts) {
  const goCache = path.join(opts.repoRoot, "apps", "voice", ".gocache");
  mkdirSync(goCache, { recursive: true });
  const env = {
    ...process.env,
    GOCACHE: goCache,
    CGO_ENABLED: "1",
    ...opts.extraEnv,
  };

  const r = spawnSync(
    "go",
    [
      "build",
      "-buildvcs=false",
      "-trimpath",
      "-o",
      opts.outFile,
      "./apps/voice/cmd/vocode-voiced",
    ],
    { cwd: opts.repoRoot, env, stdio: "inherit" },
  );
  if (r.status !== 0) {
    throw new Error(`go build failed (exit ${r.status})`);
  }
}

/**
 * @param {string} slug
 * @param {string} repoRoot
 */
export function buildVoiceNativeWindows(slug, repoRoot) {
  if (process.platform !== "win32") {
    throw new Error(`${slug} must be built on Windows`);
  }
  const host = getHostSlug();
  const winArm64Ok = slug === "win32-arm64" && forceWinArm64VoiceBuild;
  if (slug !== host && !winArm64Ok) {
    throw new Error(
      `${slug} requires a Windows ${slug === "win32-arm64" ? "arm64" : "x64"} machine (or matching CI runner). Host is ${host}.`,
    );
  }

  const cfg = configurePortAudioMsys2();
  if (!cfg.ok) {
    const subdir = msysVoiceSubdir();
    const hint =
      subdir === "clangarm64"
        ? "pacman -S mingw-w64-clang-aarch64-clang mingw-w64-clang-aarch64-portaudio mingw-w64-clang-aarch64-pkg-config"
        : "pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-portaudio mingw-w64-x86_64-pkg-config";
    throw new Error(
      `MSYS2 (${subdir}) + PortAudio not found (MSYS2_ROOT=${cfg.msysRoot}). ${hint}`,
    );
  }

  const outDir = path.join(repoRoot, "apps", "voice", "bin", slug);
  mkdirSync(outDir, { recursive: true });
  const outFile = path.join(outDir, "vocode-voiced.exe");

  console.log(`[voice-build] ${slug}: native Windows (CGO / PortAudio)`);
  goBuildVoiceNative({
    repoRoot,
    slug,
    outFile,
    extraEnv: cfg.env,
  });
}

/**
 * @param {string} slug
 * @param {string} repoRoot
 */
export function buildVoiceNativeDarwin(slug, repoRoot) {
  if (process.platform !== "darwin") {
    throw new Error(`${slug} must be built on macOS`);
  }

  const t = targetForSlug(slug, repoRoot);
  const host = getHostSlug();
  const pc = configurePortAudioDarwin();

  const outDir = path.join(repoRoot, "apps", "voice", "bin", slug);
  mkdirSync(outDir, { recursive: true });
  const outFile = path.join(outDir, "vocode-voiced");

  const extraEnv = { ...pc };
  if (slug !== host) {
    extraEnv.GOOS = "darwin";
    extraEnv.GOARCH = t.goarch;
    if (t.goarm) {
      extraEnv.GOARM = t.goarm;
    }
  }

  const mode =
    slug === host
      ? "native arch"
      : `cross GOARCH=${t.goarch}${t.goarm ? ` GOARM=${t.goarm}` : ""}`;
  console.log(`[voice-build] ${slug}: macOS ${mode} (CGO / PortAudio)`);
  goBuildVoiceNative({ repoRoot, slug, outFile, extraEnv });
}

/**
 * Host Linux, slug matches this machine (glibc).
 * @param {string} slug
 * @param {string} repoRoot
 */
export function buildVoiceNativeLinuxHost(slug, repoRoot) {
  if (process.platform !== "linux") {
    throw new Error("internal: native linux host build on non-linux");
  }
  if (slug !== getHostSlug()) {
    throw new Error(
      `internal: slug ${slug} !== host ${getHostSlug()} for native linux`,
    );
  }
  if (slug.startsWith("alpine-")) {
    throw new Error("Use Alpine Docker for alpine-* slugs");
  }

  const outDir = path.join(repoRoot, "apps", "voice", "bin", slug);
  mkdirSync(outDir, { recursive: true });
  const outFile = path.join(outDir, "vocode-voiced");

  console.log(`[voice-build] ${slug}: native Linux glibc (CGO / PortAudio)`);
  goBuildVoiceNative({ repoRoot, slug, outFile, extraEnv: {} });
}

/**
 * Build one marketplace slug (PortAudio enabled).
 * @param {string} slug
 * @param {{ repoRoot?: string }} [opts]
 */
export function buildVoiceForSlug(slug, opts = {}) {
  const repoRoot = opts.repoRoot ?? defaultRepoRoot();

  if (slug.startsWith("alpine-")) {
    buildVoiceLinuxDockerAlpine(slug, repoRoot);
    return;
  }

  if (slug.startsWith("linux-")) {
    if (process.platform === "linux" && slug === getHostSlug()) {
      buildVoiceNativeLinuxHost(slug, repoRoot);
      return;
    }
    buildVoiceLinuxDockerDebian(slug, repoRoot);
    return;
  }

  if (slug.startsWith("win32-")) {
    buildVoiceNativeWindows(slug, repoRoot);
    return;
  }

  if (slug.startsWith("darwin-")) {
    buildVoiceNativeDarwin(slug, repoRoot);
    return;
  }

  throw new Error(`Unhandled slug: ${slug}`);
}

/**
 * @param {'linux' | 'windows' | 'darwin' | null} family
 * @param {string} repoRoot
 */
export function slugsForFamily(family, repoRoot) {
  const all = loadFatTargets(repoRoot).map((t) => t.slug);
  if (!family) {
    return all;
  }
  if (family === "linux") {
    return all.filter((s) => s.startsWith("linux-") || s.startsWith("alpine-"));
  }
  if (family === "windows") {
    return all.filter((s) => s.startsWith("win32-") && s === getHostSlug());
  }
  if (family === "darwin") {
    return all.filter((s) => s.startsWith("darwin-") && s === getHostSlug());
  }
  return all;
}
