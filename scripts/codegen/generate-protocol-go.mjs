import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const root = process.cwd();
const schemaDir = path.join(root, "packages", "protocol", "schema");
const outDir = path.join(root, "packages", "protocol", "go");
const outFile = path.join(outDir, "types.generated.go");

const entries = [
  { file: "common.schema.json", def: "Anchor", name: "Anchor" },
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
  { file: "delete-file-directive.schema.json", name: "DeleteFileDirective" },
  { file: "move-path-directive.schema.json", name: "MovePathDirective" },
  {
    file: "create-folder-directive.schema.json",
    name: "CreateFolderDirective",
  },
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
    file: "voice-transcript.search-result-hit.schema.json",
    name: "VoiceTranscriptSearchHit",
  },
  {
    file: "voice-transcript.search-state.schema.json",
    name: "VoiceTranscriptWorkspaceSearchState",
  },
  {
    file: "voice-transcript.question-answer.schema.json",
    name: "VoiceTranscriptQuestionAnswer",
  },
  {
    file: "voice-transcript.clarify-resolution.schema.json",
    name: "VoiceTranscriptClarifyOffer",
  },
  {
    file: "voice-transcript.file-list-hit.schema.json",
    name: "VoiceTranscriptFileListHit",
  },
  {
    file: "voice-transcript.file-selection-state.schema.json",
    name: "VoiceTranscriptFileSearchState",
  },
  {
    file: "voice-transcript.workspace-hints.schema.json",
    name: "VoiceTranscriptWorkspaceHints",
  },
  {
    file: "voice-transcript.result.schema.json",
    name: "VoiceTranscriptCompletion",
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
    file: "host-read-file.params.schema.json",
    name: "HostReadFileParams",
  },
  {
    file: "host-read-file.result.schema.json",
    name: "HostReadFileResult",
  },
  {
    file: "host-get-document-symbols.params.schema.json",
    name: "HostGetDocumentSymbolsParams",
  },
  {
    file: "host-get-document-symbols.result.schema.json",
    name: "HostGetDocumentSymbolsResult",
  },
  {
    file: "host-workspace-symbol-search.params.schema.json",
    name: "HostWorkspaceSymbolSearchParams",
  },
  {
    file: "host-workspace-symbol-search.result.schema.json",
    name: "HostWorkspaceSymbolSearchResult",
  },
  {
    file: "command-directive.schema.json",
    name: "CommandDirective",
  },
];

const schemaCache = new Map();

function readJson(filePath) {
  if (!schemaCache.has(filePath)) {
    schemaCache.set(filePath, JSON.parse(readFileSync(filePath, "utf8")));
  }
  return schemaCache.get(filePath);
}

function getEntrySchema(entry) {
  const abs = path.join(schemaDir, entry.file);
  const schema = readJson(abs);

  if (entry.def) {
    const def = schema.$defs?.[entry.def];
    if (!def) {
      throw new Error(`Missing $defs.${entry.def} in ${entry.file}`);
    }
    return {
      schema: def,
      absPath: abs,
    };
  }

  return {
    schema,
    absPath: abs,
  };
}

function refKey(absPath, defName = null) {
  return defName ? `${absPath}#/$defs/${defName}` : absPath;
}

const namedRefMap = new Map();
for (const entry of entries) {
  const abs = path.join(schemaDir, entry.file);
  if (entry.def) {
    namedRefMap.set(refKey(abs, entry.def), entry.name);
  } else {
    namedRefMap.set(refKey(abs), entry.name);
  }
}

function resolveRef(currentAbsPath, ref) {
  const [refPathPart, fragment = ""] = ref.split("#");
  const targetAbsPath = refPathPart
    ? path.resolve(path.dirname(currentAbsPath), refPathPart)
    : currentAbsPath;

  const fragmentPath = fragment ? `#${fragment}` : "";
  const targetKey = `${targetAbsPath}${fragmentPath}`;

  const targetSchema = readJson(targetAbsPath);

  if (!fragment) {
    return {
      absPath: targetAbsPath,
      schema: targetSchema,
      key: targetKey,
    };
  }

  const parts = fragment.replace(/^\//, "").split("/").map(unescapeJsonPointer);
  let node = targetSchema;
  for (const part of parts) {
    if (!(part in node)) {
      throw new Error(`Could not resolve ref ${ref} from ${currentAbsPath}`);
    }
    node = node[part];
  }

  return {
    absPath: targetAbsPath,
    schema: node,
    key: targetKey,
  };
}

function unescapeJsonPointer(segment) {
  return segment.replace(/~1/g, "/").replace(/~0/g, "~");
}

function toGoTypeName(name) {
  return name
    .replace(/[^a-zA-Z0-9]+/g, " ")
    .trim()
    .split(/\s+/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

function toGoFieldName(jsonName) {
  const base = toGoTypeName(jsonName);
  if (base === "Id") return "ID";
  if (base === "Url") return "URL";
  if (base === "Jsonrpc") return "JSONRPC";
  return base;
}

function schemaToGoType(schema, currentAbsPath, ctx) {
  if (schema.$ref) {
    const resolved = resolveRef(currentAbsPath, schema.$ref);

    if (namedRefMap.has(resolved.key)) {
      return namedRefMap.get(resolved.key);
    }

    return schemaToGoType(resolved.schema, resolved.absPath, ctx);
  }

  if (schema.oneOf) {
    if (schema.oneOf.length === 1) {
      return schemaToGoType(schema.oneOf[0], currentAbsPath, ctx);
    }
    const mergedObjectSchema = mergeOneOfObjectSchema(schema, currentAbsPath);
    return schemaToGoType(mergedObjectSchema, currentAbsPath, ctx);
  }

  if (schema.type === "string" || schema.const !== undefined) {
    return "string";
  }

  if (schema.type === "integer") {
    return "int64";
  }

  if (schema.type === "number") {
    return "float64";
  }

  if (schema.type === "boolean") {
    return "bool";
  }

  if (schema.type === "array") {
    if (!schema.items) {
      throw new Error(`Array missing items in ${ctx.name}`);
    }
    return `[]${schemaToGoType(schema.items, currentAbsPath, ctx)}`;
  }

  if (schema.type === "object") {
    const properties = schema.properties ?? {};
    const required = new Set(schema.required ?? []);
    const propEntries = Object.entries(properties);

    if (propEntries.length === 0) {
      return "struct{}";
    }

    const lines = ["struct {"];

    for (const [jsonName, propSchema] of propEntries) {
      const fieldName = toGoFieldName(jsonName);
      const goType = schemaToGoType(propSchema, currentAbsPath, ctx);
      const isRequired = required.has(jsonName);
      const fieldType = isRequired ? goType : toOptionalGoType(goType);
      const omitempty = isRequired ? "" : ",omitempty";
      lines.push(
        `	${fieldName} ${fieldType} \`json:"${jsonName}${omitempty}"\``,
      );
    }

    lines.push("}");
    return lines.join("\n");
  }

  throw new Error(
    `Unsupported schema in ${ctx.name}: ${JSON.stringify(schema, null, 2)}`,
  );
}

function toOptionalGoType(goType) {
  if (
    goType === "string" ||
    goType === "int" ||
    goType === "float64" ||
    goType === "bool" ||
    goType === "interface{}" ||
    goType.startsWith("[]")
  ) {
    return goType;
  }

  if (goType.startsWith("*")) {
    return goType;
  }

  return `*${goType}`;
}

/**
 * Resolves $ref and flattens `allOf` into a single `type: "object"` schema so oneOf variants
 * can compose shared `$defs` (same pattern as schema files using allOf + $ref).
 */
function flattenToObjectSchema(schema, currentAbsPath) {
  if (!schema || typeof schema !== "object") {
    throw new Error("Invalid schema node for flattenToObjectSchema");
  }
  if (schema.$ref) {
    const resolved = resolveRef(currentAbsPath, schema.$ref);
    return flattenToObjectSchema(resolved.schema, resolved.absPath);
  }
  if (Array.isArray(schema.allOf)) {
    const merged = {
      type: "object",
      properties: {},
      required: [],
      additionalProperties: schema.additionalProperties,
    };
    for (const part of schema.allOf) {
      const flat = flattenToObjectSchema(part, currentAbsPath);
      if (flat.type !== "object") {
        throw new Error(
          "Unsupported allOf: member must flatten to type object (check voice-transcript.params oneOf variants)",
        );
      }
      Object.assign(merged.properties, flat.properties ?? {});
      for (const key of flat.required ?? []) {
        if (!merged.required.includes(key)) {
          merged.required.push(key);
        }
      }
      if (flat.additionalProperties === false) {
        merged.additionalProperties = false;
      }
    }
    return merged;
  }
  if (schema.type === "object") {
    return {
      type: "object",
      properties: { ...(schema.properties ?? {}) },
      required: [...(schema.required ?? [])],
      additionalProperties: schema.additionalProperties,
    };
  }
  throw new Error(
    `Expected object or allOf for oneOf variant, got: ${JSON.stringify(schema).slice(0, 240)}`,
  );
}

function mergeOneOfObjectSchema(schema, currentAbsPath) {
  const variants = schema.oneOf.map((variant) =>
    flattenToObjectSchema(variant, currentAbsPath),
  );

  const objectVariants = variants.filter(
    (variant) => variant.type === "object",
  );
  if (
    objectVariants.length !== variants.length ||
    objectVariants.length === 0
  ) {
    throw new Error("Unsupported oneOf union: expected object variants only");
  }

  const mergedProperties = {};
  const requiredIntersection = new Set(objectVariants[0].required ?? []);
  for (const variant of objectVariants) {
    const props = variant.properties ?? {};
    for (const [name, value] of Object.entries(props)) {
      if (!(name in mergedProperties)) {
        mergedProperties[name] = value;
      }
    }

    const requiredSet = new Set(variant.required ?? []);
    for (const key of [...requiredIntersection]) {
      if (!requiredSet.has(key)) {
        requiredIntersection.delete(key);
      }
    }
  }

  return {
    type: "object",
    properties: mergedProperties,
    required: [...requiredIntersection],
  };
}

function emitEntry(entry) {
  const { schema, absPath } = getEntrySchema(entry);
  const ctx = { name: entry.name };

  if (schema.oneOf) {
    if (schema.oneOf.length === 1) {
      const targetType = schemaToGoType(schema.oneOf[0], absPath, ctx);
      return `type ${entry.name} = ${targetType}`;
    }
    const mergedObjectSchema = mergeOneOfObjectSchema(schema, absPath);
    const goType = schemaToGoType(mergedObjectSchema, absPath, ctx);
    return `type ${entry.name} ${goType}`;
  }

  const goType = schemaToGoType(schema, absPath, ctx);

  if (goType.startsWith("struct {\n")) {
    return `type ${entry.name} ${goType}`;
  }

  return `type ${entry.name} ${goType}`;
}

mkdirSync(outDir, { recursive: true });

const generatedTypes = entries.map(emitEntry).join("\n\n");

const jsonRpcHelpers = `type JSONRPCRequest[T any] struct {
\tJSONRPC string \`json:"jsonrpc"\`
\tID      int64  \`json:"id"\`
\tMethod  string \`json:"method"\`
\tParams  T      \`json:"params"\`
}

type JSONRPCResponse[T any] struct {
\tJSONRPC string \`json:"jsonrpc"\`
\tID      int64  \`json:"id"\`
\tResult  T      \`json:"result"\`
}

type JSONRPCErrorObject struct {
\tCode    int    \`json:"code"\`
\tMessage string \`json:"message"\`
}

type JSONRPCErrorResponse struct {
\tJSONRPC string            \`json:"jsonrpc"\`
\tID      *int64            \`json:"id"\`
\tError   JSONRPCErrorObject \`json:"error"\`
}
`;

const output = `// AUTO-GENERATED. DO NOT EDIT.

package protocol

${generatedTypes}

${jsonRpcHelpers}
`;

writeFileSync(outFile, output, "utf8");

console.log(`Generated ${path.relative(root, outFile)}`);
