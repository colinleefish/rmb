// Static-export the UI and copy it into the Go binary's embed dir.
//
// The dev-only proxy route (src/app/api) is a dynamic route handler that a
// static export cannot render, so we temporarily move it aside during the
// export and always restore it afterwards.
//
// Env:
//   EMBED_TARGET   destination dir for the export. Defaults to the Go embed
//                  dir (../internal/http/static/web). Set to "skip" to leave
//                  the output in ./out only (used by the Docker build).
import { execSync } from "node:child_process";
import {
  cpSync,
  existsSync,
  mkdirSync,
  readdirSync,
  renameSync,
  rmSync,
} from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const uiDir = path.resolve(scriptDir, "..");
const apiDir = path.join(uiDir, "src/app/api");
const apiStash = path.join(uiDir, ".embed-stash-api");
const outDir = path.join(uiDir, "out");

const target =
  process.env.EMBED_TARGET === "skip"
    ? null
    : process.env.EMBED_TARGET
      ? path.resolve(process.env.EMBED_TARGET)
      : path.resolve(uiDir, "../internal/http/static/web");

const hadApi = existsSync(apiDir);
if (hadApi) {
  rmSync(apiStash, { recursive: true, force: true });
  renameSync(apiDir, apiStash);
}

try {
  rmSync(outDir, { recursive: true, force: true });
  execSync("pnpm exec next build", {
    cwd: uiDir,
    stdio: "inherit",
    env: { ...process.env, NEXT_EXPORT: "1" },
  });
} finally {
  if (hadApi) {
    rmSync(apiDir, { recursive: true, force: true });
    renameSync(apiStash, apiDir);
  }
}

if (target) {
  rmSync(target, { recursive: true, force: true });
  mkdirSync(target, { recursive: true });
  for (const entry of readdirSync(outDir)) {
    cpSync(path.join(outDir, entry), path.join(target, entry), {
      recursive: true,
    });
  }
  console.log(`\nEmbedded static export → ${target}`);
}
