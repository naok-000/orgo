import { defineConfig } from "vitest/config";

export default defineConfig({
  build: {
    outDir: "../internal/server/dist",
    emptyOutDir: true,
    // dist/ is committed and go:embed'd into the binary; skip sourcemaps so
    // they don't bloat the repo/binary. Use `vite build` locally (without
    // this config override) or `vite dev` for a source-mapped build.
    sourcemap: false,
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:35911",
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
    restoreMocks: true,
  },
});
