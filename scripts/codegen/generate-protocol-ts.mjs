import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import $RefParser from "@apidevtools/json-schema-ref-parser";
import { compile } from "json-schema-to-typescript";

const root = process.cwd();
const schemaDir = path.join(root, "packages", "protocol", "schema");
const outDir = path.join(root, "packages", "protocol", "typescript");
const outFile = path.join(outDir, "types.generated.ts");

const entries = [
  {
    file: "edit-action.replace-between-anchors.schema.json",
    name: "ReplaceBetweenAnchorsAction",
  },
  {
    file: "edit-action.create-file.schema.json",
    name: "CreateFileAction",
  },
  {
    file: "edit-action.append-to-file.schema.json",
    name: "AppendToFileAction",
  },
  { file: "edit-action.schema.json", name: "EditAction" },
  { file: "ping.params.schema.json", name: "PingParams" },
  { file: "ping.result.schema.json", name: "PingResult" },
  { file: "edit-directive.schema.json", name: "EditDirective" },
  { file: "undo-directive.schema.json", name: "UndoDirective" },
  {
    file: "voice-transcript.params.schema.json",
    name: "VoiceTranscriptParams",
  },
  {
    file: "voice-transcript.directive-apply-item.schema.json",
    name: "VoiceTranscriptDirectiveApplyItem",
  },
  {
    file: "navigation-action.schema.json",
    name: "NavigationAction",
  },
  {
    file: "navigation-directive.schema.json",
    name: "NavigationDirective",
  },
  {
    file: "voice-transcript.directive.schema.json",
    name: "VoiceTranscriptDirective",
  },
  {
    file: "voice-transcript.result.schema.json",
    name: "VoiceTranscriptResult",
  },
  {
    file: "host-apply.params.schema.json",
    name: "HostApplyParams",
  },
  {
    file: "host-apply.result.schema.json",
    name: "HostApplyResult",
  },
  {
    file: "command-directive.schema.json",
    name: "CommandDirective",
  },
];

mkdirSync(outDir, { recursive: true });

const compilerOptions = {
  bannerComment: "",
  style: {
    singleQuote: false,
    semi: true,
  },
};

const typeMap = new Map(); // name → definition

function extractTypes(ts) {
  const lines = ts.split("\n");
  const results = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    const interfaceMatch = line.match(/^export interface (\w+)\b/);
    const typeMatch = line.match(/^export type (\w+)\b/);

    if (!interfaceMatch && !typeMatch) {
      i += 1;
      continue;
    }

    const name = interfaceMatch?.[1] ?? typeMatch?.[1];
    const isTypeAlias = Boolean(typeMatch);
    const block = [line];

    // Single-line declaration like:
    // export interface PingParams {}
    // export type EditAction = ReplaceBetweenAnchorsAction;
    if (line.includes("{}") || line.trim().endsWith(";")) {
      results.push({
        name,
        code: block.join("\n"),
      });
      i += 1;
      continue;
    }

    // Multi-line declaration
    let braceDepth =
      (line.match(/{/g) ?? []).length - (line.match(/}/g) ?? []).length;

    i += 1;

    while (i < lines.length) {
      const nextLine = lines[i];
      block.push(nextLine);

      braceDepth +=
        (nextLine.match(/{/g) ?? []).length -
        (nextLine.match(/}/g) ?? []).length;

      // Type aliases can include union members that close braces before the
      // declaration is complete, so always wait for the trailing semicolon.
      if (isTypeAlias) {
        if (braceDepth <= 0 && nextLine.trim().endsWith(";")) {
          break;
        }
      } else if (braceDepth <= 0) {
        break;
      }

      i += 1;
    }

    results.push({
      name,
      code: block.join("\n"),
    });

    i += 1;
  }

  return results;
}

for (const entry of entries) {
  const schemaPath = path.join(schemaDir, entry.file);
  const dereferenced = await $RefParser.dereference(schemaPath);

  const ts = await compile(dereferenced, entry.name, compilerOptions);

  const types = extractTypes(ts);

  console.log(
    `[codegen-ts] ${entry.name}: extracted ${types.map((t) => t.name).join(", ")}`,
  );

  for (const t of types) {
    if (!typeMap.has(t.name)) {
      typeMap.set(t.name, t.code.trim());
    }
  }
}

const header = "// AUTO-GENERATED. DO NOT EDIT.\n";

const output = [header, ...typeMap.values()].join("\n\n");

writeFileSync(outFile, output, "utf8");

console.log(`Generated ${path.relative(root, outFile)}`);
