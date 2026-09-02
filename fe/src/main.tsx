import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Providers } from "./app/providers";
import { initSentry } from "./shared/lib/sentry";
import { initAnalyticsIfConsented } from "./shared/lib/consent";
import "./index.css";

initSentry();
// GA4/PostHog start only after the user opts in via the cookie banner
// (consent.ts) — returning visitors who accepted get them right away.
initAnalyticsIfConsented();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Providers />
  </StrictMode>,
);
