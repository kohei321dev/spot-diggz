import { readFile, writeFile } from "node:fs/promises";

const [inputPath, outputPath] = process.argv.slice(2);
if (!inputPath || !outputPath) {
  throw new Error("usage: node scripts/prepare-e2e-catalog.mjs <input> <output>");
}

const catalog = JSON.parse(await readFile(inputPath, "utf8"));
if (!Array.isArray(catalog.facilities) || catalog.facilities.length === 0) {
  throw new Error("E2E catalog must contain at least one facility");
}

const verifiedAt = new Date(Date.now() - 60_000).toISOString();
for (const facility of catalog.facilities) {
  facility.verifiedAt = verifiedAt;
  facility.dynamicVerifiedAt = verifiedAt;
  facility.stableVerifiedAt = verifiedAt;
}

await writeFile(outputPath, `${JSON.stringify(catalog, null, 2)}\n`, { mode: 0o600 });
