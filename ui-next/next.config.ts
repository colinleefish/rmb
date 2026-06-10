import type { NextConfig } from "next";

// Two build modes:
//  - dev (`next dev`): normal app + the /api/v1 proxy route handler (which
//    injects Basic auth for the remote backend during local development).
//  - embed (`NEXT_EXPORT=1`, via `pnpm build:embed`): a static export served by
//    the Go binary under /ui. Same-origin, so no proxy/auth injection is needed
//    (the Go server serves both the UI and /api/v1, and handles auth itself).
const EXPORT = process.env.NEXT_EXPORT === "1";

const nextConfig: NextConfig = EXPORT
  ? {
      output: "export",
      basePath: "/ui",
      trailingSlash: true,
      images: { unoptimized: true },
    }
  : {};

export default nextConfig;
