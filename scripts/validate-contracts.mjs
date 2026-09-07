import assert from "node:assert/strict";
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

const openAPIPath = "docs/specifications/facility-catalog.openapi.yaml";
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
  "/api/facilities/search",
  "/api/locations/search",
  "/api/recommendations",
  "/api/corrections",
  "/api/events",
  "/integrations/slack/commands",
  "/integrations/discord/interactions",
  "/metrics",
]);
const actualPaths = new Set(Object.keys(openAPI.paths ?? {}));
if (actualPaths.size !== expectedPaths.size || [...expectedPaths].some((path) => !actualPaths.has(path))) {
  throw new Error(`OpenAPI paths do not match the application routes: ${[...actualPaths].join(", ")}`);
}

const nearbyInput = openAPI.components.schemas.NearbySearchInput;
if (nearbyInput.additionalProperties !== false ||
    nearbyInput.required.join(",") !== "query" ||
    nearbyInput.properties.limit.default !== 5 || nearbyInput.properties.limit.maximum !== 10 ||
    nearbyInput.properties.radiusKm.default !== 10 || nearbyInput.properties.radiusKm.minimum !== 0.1 || nearbyInput.properties.radiusKm.maximum !== 50 ||
    nearbyInput.properties.genre.enum.join(",") !== "skatepark,street" ||
    nearbyInput.properties.sort.enum.join(",") !== "distance" ||
    !openAPI.paths["/api/facilities/search"].post.security?.[0]?.apiBearer) {
  throw new Error("Nearby read API contract differs from DR-0021");
}

const facilitySchema = openAPI.components.schemas.Facility;
const prefectureSource = await readFile("internal/facility/prefectures.go", "utf8");
const prefectureConstant = prefectureSource.match(/^const JapanesePrefectures = "([^"]+)"\r?$/m);
assert.ok(prefectureConstant, "GoのJapanesePrefectures定数が見つかりません");
const prefectures = prefectureConstant[1].split(" ");
assert.equal(prefectures.length, 47, "国内住所の受理対象は47都道府県です");
assert.equal(new Set(prefectures).size, 47, "都道府県の定義に重複があります");
assert.deepEqual(facilitySchema.properties.prefecture.enum, prefectures,
  "OpenAPIの都道府県とGoのvalidator定義が一致しません");

// Facilityはレスポンス契約。catalog入力で受理する空配列・空文字も、非knownの出力では省略される。
const hoursStatuses = ["known", "not_applicable", "unknown"];
assert.deepEqual(facilitySchema.properties.hoursStatus.enum, hoursStatuses);
assert.ok(!facilitySchema.required.includes("hours"), "hoursはknown時だけ必須です");
assert.ok(!facilitySchema.required.includes("hoursStatus"), "旧recordのhoursStatus省略を保持します");
assert.deepEqual(facilitySchema.oneOf.map((branch) => branch.properties.hoursStatus.const), hoursStatuses);
const [knownHours, noHoursSystem, unknownHours] = facilitySchema.oneOf;
assert.ok(knownHours.required.includes("hours"));
assert.ok(!knownHours.required.includes("hoursStatus"), "省略した旧recordはknown分岐で受理します");
assert.equal(knownHours.properties.hours.minItems, 1);
assert.equal(facilitySchema.properties.hours.items.$ref, "#/components/schemas/OperatingHours");

for (const branch of [noHoursSystem, unknownHours]) {
  for (const field of ["hoursStatus", "genre", "generalUseStatus", "availabilityNote", "englishTranslation"]) {
    assert.ok(branch.required.includes(field), `非knownの必須項目がありません: ${field}`);
  }
  assert.equal(branch.properties.availabilityNote.pattern, "\\S");
  assert.ok(branch.properties.englishTranslation.required.includes("availabilityNote"));
  assert.equal(branch.properties.englishTranslation.properties.availabilityNote.pattern, "\\S");
  assert.deepEqual(branch.not.anyOf.map((condition) => condition.required), [["hours"], ["hoursBasis"]],
    "非knownのレスポンスはhoursとhoursBasisを省略します");
}
assert.equal(noHoursSystem.properties.genre.const, "street");
assert.ok(noHoursSystem.required.includes("hoursSourceUrl"));
assert.deepEqual(noHoursSystem.properties.generalUseStatus.enum, ["regular", "limited"]);
assert.equal(unknownHours.properties.generalUseStatus.const, "schedule_check_required");
assert.deepEqual(facilitySchema.properties.genre.enum, ["skatepark", "street"]);
assert.deepEqual(facilitySchema.dependentRequired.genre, ["skatingPermissionSourceUrl"]);
const scheduleCheck = facilitySchema.allOf.find((condition) =>
  condition.if?.properties?.generalUseStatus?.const === "schedule_check_required");
assert.ok(scheduleCheck?.if?.required?.includes("generalUseStatus"));
assert.ok(scheduleCheck?.then?.required?.includes("availabilityNote"));
assert.equal(scheduleCheck.then.properties.availabilityNote.pattern, "\\S");
assert.ok(!openAPI.components.schemas.FacilityEnglishTranslation.required.includes("availabilityNote"),
  "旧known recordの英語availabilityNoteは任意です");

const hoursSource = facilitySchema.properties.hoursSourceUrl;
assert.equal(hoursSource.type, "string");
assert.equal(hoursSource.format, "uri");
const hoursSourcePattern = new RegExp(hoursSource.pattern);
for (const url of ["https://example.com", "https://example.com/rules", "https://[::1]/rules"]) {
  assert.ok(hoursSourcePattern.test(url), "営業時間のHTTPS出典を表現できません");
}
for (const url of ["http://example.com", "https:///rules", "https://user@example.com/rules", "https://example.com:443/rules", "https://example.com:/rules", "https://[::1]:/rules", "https://[]/rules"]) {
  assert.ok(!hoursSourcePattern.test(url), "営業時間の出典は認証情報・明示的portなしのHTTPSが必要です");
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

console.log(`Contracts OK: ${jsonFiles.length} JSON files, ${actualPaths.size} API paths, ${references.length} local refs, ${prefectures.length} prefectures, ${hoursStatuses.length} hours states`);
