import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const root = process.cwd();
const schemaDir = path.join(root, "packages", "protocol", "schema");
const outDir = path.join(root, "packages", "protocol", "go");
const outFile = path.join(outDir, "types.generated.go");

const entries = [
  { file: "common.schema.json", def: "Anchor", name: "Anchor" },
  { file: "common.schema.json", def: "EditFailure", name: "EditFailure" },
  {
    file: "edit-action.replace-between-anchors.schema.json",
    name: "ReplaceBetweenAnchorsAction",
  },
  { file: "edit-action.schema.json", name: "EditAction" },
  { file: "ping.params.schema.json", name: "PingParams" },
  { file: "ping.result.schema.json", name: "PingResult" },
  { file: "edit-apply.params.schema.json", name: "EditApplyParams" },
  { file: "edit-apply.result.schema.json", name: "EditApplyResult" },
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

    throw new Error(
      `Unsupported oneOf with ${schema.oneOf.length} variants in ${ctx.name}`,
    );
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

function emitEntry(entry) {
  const { schema, absPath } = getEntrySchema(entry);
  const ctx = { name: entry.name };

  if (schema.oneOf) {
    if (schema.oneOf.length === 1) {
      const targetType = schemaToGoType(schema.oneOf[0], absPath, ctx);
      return `type ${entry.name} = ${targetType}`;
    }

    throw new Error(`Unsupported union root for ${entry.name}`);
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
