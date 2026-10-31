/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { sentryVitePlugin } from "@sentry/vite-plugin";
import path from "path";
import sitemapPlugin from "./plugins/vite-plugin-sitemap";

// Only the Docker build in CI has this. Everywhere else, the test job's build
// included, there is no upload and therefore no source maps at all.
const sentryAuthToken = process.env.SENTRY_AUTH_TOKEN;

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    sitemapPlugin(),
    ...(sentryAuthToken
      ? [
          sentryVitePlugin({
            org: "fluxlab",
            project: "jobber-fe",
            authToken: sentryAuthToken,
            telemetry: false,
            release: { name: process.env.VITE_SENTRY_RELEASE },
            // The image serves dist/ as is, so the maps must not outlive the upload.
            sourcemaps: { filesToDeleteAfterUpload: ["./dist/**/*.map"] },
            // Payments are live: a Sentry outage costs readable stack traces for
            // one release, and must not hold that release back.
            errorHandler: (error) =>
              console.warn("Sentry source map upload failed:", error),
          }),
        ]
      : []),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    cssCodeSplit: true,
    sourcemap: sentryAuthToken ? "hidden" : false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          if (id.includes("react-router") || id.includes("@remix-run"))
            return "router";
          if (
            id.includes("/react/") ||
            id.includes("/react-dom/") ||
            id.includes("/scheduler/")
          )
            return "react";
          if (id.includes("@sentry")) return "sentry";
          if (id.includes("@tanstack")) return "query";
          if (id.includes("@dnd-kit")) return "dnd";
          if (id.includes("i18next")) return "i18n";
          if (id.includes("posthog-js")) return "posthog";
          if (id.includes("lucide-react") || id.includes("sonner"))
            return "ui-icons";
          // @hookform/resolvers has to land here too: its `zod` entry already
          // matched "zod" and went into `forms`, while the shared root it
          // imports stayed in `vendor` and itself imports react-hook-form —
          // that closed a forms -> vendor -> forms chunk cycle.
          if (
            id.includes("zod") ||
            id.includes("react-hook-form") ||
            id.includes("@hookform")
          )
            return "forms";
          if (id.includes("date-fns")) return "date";
          if (id.includes("dompurify") || id.includes("marked"))
            return "markdown";
          return "vendor";
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    // `api.ts` falls back to a relative base URL, which ky cannot parse outside a browser.
    env: { VITE_API_BASE_URL: "http://localhost:3000/api/v1" },
    setupFiles: ["./src/test/setup.ts"],
    include: [
      "src/**/*.test.{ts,tsx}",
      "plugins/**/*.test.ts",
      // Build scripts are plain ESM and outside tsc's program, but the
      // contract they share with the app (the prerenderer's seeded cookie
      // decision) still has to be checked against the app's own reader.
      "scripts/**/*.test.mjs",
    ],
    coverage: {
      provider: "v8",
      reporter: ["text", "text-summary"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.test.{ts,tsx}", "src/test/**"],
    },
  },
});
