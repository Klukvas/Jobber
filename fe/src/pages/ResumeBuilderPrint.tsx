import { useEffect, useRef, useState } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { TEMPLATE_MAP } from "@/features/resume-builder/lib/templateRegistry";
import { ProfessionalTemplate } from "@/features/resume-builder/components/preview/ProfessionalTemplate";
import type { FullResumeDTO } from "@/shared/types/resume-builder";

// Google Fonts that need to be loaded dynamically.
// System fonts (Georgia, Arial, Times New Roman) need no loading.
const GOOGLE_FONTS: Record<string, string> = {
  Roboto: "Roboto:wght@400;500;700",
  "Open Sans": "Open+Sans:wght@400;600;700",
  Lato: "Lato:wght@400;700",
  Montserrat: "Montserrat:wght@400;500;700",
  Poppins: "Poppins:wght@400;500;700",
  Inter: "Inter:wght@400;500;700",
  Merriweather: "Merriweather:wght@400;700",
  "PT Serif": "PT+Serif:wght@400;700",
  "Source Sans Pro": "Source+Sans+Pro:wght@400;600;700",
  Nunito: "Nunito:wght@400;600;700",
  Raleway: "Raleway:wght@400;500;700",
  "Playfair Display": "Playfair+Display:wght@400;700",
};

declare global {
  interface Window {
    __RESUME_DATA__?: FullResumeDTO;
    __PDF_READY__?: boolean;
  }
}

function loadGoogleFont(fontFamily: string): Promise<void> {
  const spec = GOOGLE_FONTS[fontFamily];
  if (!spec) return Promise.resolve(); // system font — nothing to load

  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = `https://fonts.googleapis.com/css2?family=${spec}&display=swap`;
  document.head.appendChild(link);

  const fontTimeout = new Promise<void>((resolve) => setTimeout(resolve, 5000));
  return Promise.race([
    document.fonts.ready.then(() => undefined),
    fontTimeout,
  ]);
}

function signalReady() {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      window.__PDF_READY__ = true;
    });
  });
}

export default function ResumeBuilderPrint() {
  const setResume = useResumeBuilderStore((s) => s.setResume);
  const [injectedResume, setInjectedResume] = useState<FullResumeDTO | null>(
    null,
  );
  const [ready, setReady] = useState(false);
  const injectedRef = useRef(false);

  const [timedOut, setTimedOut] = useState(false);

  // Poll for data injected by Rod's page.Eval()
  useEffect(() => {
    let cancelled = false;
    const poll = setInterval(() => {
      if (cancelled || injectedRef.current) return;
      const data = window.__RESUME_DATA__;
      if (data) {
        clearInterval(poll);
        injectedRef.current = true;
        setInjectedResume(data);
        setResume(data);
      }
    }, 50);
    const timeout = setTimeout(() => {
      if (!injectedRef.current) {
        clearInterval(poll);
        setTimedOut(true);
      }
    }, 30000);
    return () => {
      cancelled = true;
      clearInterval(poll);
      clearTimeout(timeout);
    };
  }, [setResume]);

  useEffect(() => {
    if (!injectedResume) return;
    loadGoogleFont(injectedResume.font_family)
      .then(() => {
        setReady(true);
        signalReady();
      })
      .catch(() => {
        setReady(true);
        signalReady();
      });
  }, [injectedResume]);

  if (timedOut) {
    return <div id="print-error">Timeout waiting for resume data</div>;
  }

  if (!injectedResume || !ready) {
    return <div id="print-loading" />;
  }

  const TemplateComponent =
    TEMPLATE_MAP[injectedResume.template_id] ?? ProfessionalTemplate;

  // Exact same styling as PreviewPanel — zoom scales everything (rem, px, borders).
  const fontZoom =
    injectedResume.font_size > 0 ? injectedResume.font_size / 12 : 1;
  const lineHeight =
    injectedResume.spacing > 0 ? injectedResume.spacing / 100 : 1.15;

  return (
    <>
      <style>{`
        @page { size: A4; margin: 0; }
        html, body { margin: 0; padding: 0; background: white; }

        /* Chrome's print engine cannot split flex containers across pages.
           The entire flex block is treated as one atomic unit — if it exceeds
           one page, it jumps entirely to page 2 (or overflows).
           Fix: convert TwoColumnLayout's flex to float-based layout. */
        .min-h-full {
          display: block !important;
          min-height: auto !important;
        }
        .min-h-full > div {
          float: left;
        }
        .min-h-full::after {
          content: '';
          display: block;
          clear: both;
        }

        [data-avoid-break] { break-inside: avoid; }
      `}</style>
      <div
        className="text-black"
        style={{
          width: "210mm",
          boxSizing: "border-box",
          padding: `${injectedResume.margin_top}px ${injectedResume.margin_right}px ${injectedResume.margin_bottom}px ${injectedResume.margin_left}px`,
        }}
      >
        <div
          style={{
            fontFamily: injectedResume.font_family,
            lineHeight,
            zoom: fontZoom,
          }}
        >
          <TemplateComponent editable={false} />
        </div>
      </div>
    </>
  );
}
