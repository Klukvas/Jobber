import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Providers } from "./app/providers";
import { initSentry } from "./shared/lib/sentry";
import { initGA4 } from "./shared/lib/gtag";
import { initPostHog } from "./shared/lib/posthog";
import "./index.css";

initSentry();
initGA4();
initPostHog();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Providers />
  </StrictMode>,
);
