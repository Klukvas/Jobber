/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import sitemapPlugin from "./plugins/vite-plugin-sitemap";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), sitemapPlugin()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    cssCodeSplit: true,
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
          if (id.includes("zod") || id.includes("react-hook-form"))
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
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "text-summary"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/**/*.test.{ts,tsx}", "src/test/**"],
    },
  },
});
