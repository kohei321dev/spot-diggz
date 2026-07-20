import { readFile } from "node:fs/promises";
import { parseDocument } from "yaml";

const jsonFiles = [
  "data/facilities.json",
  "data/facility-candidates.json",
  "testdata/facilities.dev.json",
];

for (const path of jsonFiles) {
  JSON.parse(await readFile(path, "utf8"));
}

const openAPIPath = "docs/api/facility-catalog.openapi.yaml";
const source = await readFile(openAPIPath, "utf8");
const document = parseDocument(source, { strict: true, uniqueKeys: true });
if (document.errors.length > 0) {
  throw new Error(document.errors.map((error) => error.message).join("\n"));
}
const openAPI = document.toJS();

const expectedPaths = new Set([
  "/healthz",
  "/readyz",
  "/api/facilities",
  "/api/facilities/{facilityId}",
  "/api/locations/search",
  "/api/recommendations",
  "/api/corrections",
  "/api/events",
  "/metrics",
]);
const actualPaths = new Set(Object.keys(openAPI.paths ?? {}));
if (actualPaths.size !== expectedPaths.size || [...expectedPaths].some((path) => !actualPaths.has(path))) {
  throw new Error(`OpenAPI paths do not match the application routes: ${[...actualPaths].join(", ")}`);
}

const references = [];
function collectReferences(value) {
  if (Array.isArray(value)) {
    value.forEach(collectReferences);
    return;
  }
  if (!value || typeof value !== "object") {
    return;
  }
  if (typeof value.$ref === "string") {
    references.push(value.$ref);
  }
  Object.values(value).forEach(collectReferences);
}
collectReferences(openAPI);

for (const reference of references) {
  if (!reference.startsWith("#/")) {
    throw new Error(`Only local OpenAPI references are allowed: ${reference}`);
  }
  let current = openAPI;
  for (const rawPart of reference.slice(2).split("/")) {
    const part = rawPart.replaceAll("~1", "/").replaceAll("~0", "~");
    if (!current || typeof current !== "object" || !(part in current)) {
      throw new Error(`OpenAPI reference does not resolve: ${reference}`);
    }
    current = current[part];
  }
}

console.log(`Contracts OK: ${jsonFiles.length} JSON files, ${actualPaths.size} API paths, ${references.length} local refs`);
