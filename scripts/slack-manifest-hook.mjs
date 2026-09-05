import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const manifestPath = path.resolve(scriptDirectory, "..", "slack-manifest.json");
const manifest = await readFile(manifestPath, "utf8");

JSON.parse(manifest);
process.stdout.write(manifest);

