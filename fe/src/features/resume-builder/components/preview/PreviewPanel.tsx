import { useRef, useState, useEffect, useCallback } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { TEMPLATE_MAP } from "../../lib/templateRegistry";
import { ProfessionalTemplate } from "./ProfessionalTemplate";

const A4_WIDTH_PX = 793; // 210mm at 96dpi
const A4_HEIGHT_PX = 1122; // 297mm at 96dpi

/** Minimum space (px) a section needs at the bottom of a page to stay on it */
const MIN_SECTION_HEIGHT = 80;

interface PageBreak {
  /** Offset from the top of the measurer where this page starts */
  readonly start: number;
  /** Height of the white-space mask at the bottom of this page card */
  readonly whiteSpace: number;
}

// ---------------------------------------------------------------------------
// Smart page-break calculation
// ---------------------------------------------------------------------------

function calculatePageBreaks(measureEl: HTMLElement): readonly PageBreak[] {
  const totalHeight = measureEl.scrollHeight;
  if (totalHeight <= A4_HEIGHT_PX) {
    return [{ start: 0, whiteSpace: 0 }];
  }

  const containerRect = measureEl.getBoundingClientRect();

  // Collect vertical positions of all section wrappers
  const sectionPositions: number[] = [];
  measureEl.querySelectorAll("[data-avoid-break]").forEach((el) => {
    const pos = el.getBoundingClientRect().top - containerRect.top;
    sectionPositions.push(pos);
  });
  sectionPositions.sort((a, b) => a - b);

  const pageStarts: number[] = [0];
  let currentStart = 0;

  while (currentStart + A4_HEIGHT_PX < totalHeight) {
    let pageEnd = currentStart + A4_HEIGHT_PX;
    const dangerZoneStart = pageEnd - MIN_SECTION_HEIGHT;

    // Find sections whose top falls in the danger zone of this page
    const sectionsInDanger = sectionPositions.filter(
      (pos) => pos > dangerZoneStart && pos < pageEnd && pos > currentStart,
    );

    if (sectionsInDanger.length > 0) {
      // Push the page break to just before the earliest section in the zone
      pageEnd = Math.min(...sectionsInDanger);
    }

    pageStarts.push(pageEnd);
    currentStart = pageEnd;
  }

  return pageStarts.map((start, i) => {
    const nextStart =
      i < pageStarts.length - 1 ? pageStarts[i + 1] : totalHeight;
    const contentHeight = nextStart - start;
    const whiteSpace = Math.max(0, A4_HEIGHT_PX - contentHeight);
    return { start, whiteSpace };
  });
}

function pagesEqual(a: readonly PageBreak[], b: readonly PageBreak[]): boolean {
  if (a.length !== b.length) return false;
  return a.every(
    (p, i) => p.start === b[i].start && p.whiteSpace === b[i].whiteSpace,
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface PreviewPanelProps {
  readonly editable?: boolean;
}

export function PreviewPanel({ editable = false }: PreviewPanelProps) {
  const resume = useResumeBuilderStore((s) => s.resume);
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(editable ? 1 : 0.65);
  const [pages, setPages] = useState<readonly PageBreak[]>([
    { start: 0, whiteSpace: 0 },
  ]);

  const updateScale = useCallback(() => {
    if (!editable || !containerRef.current) return;
    const available = containerRef.current.clientWidth - 48; // 24px padding each side
    const newScale = Math.min(available / A4_WIDTH_PX, 1);
    setScale(Math.max(newScale, 0.5));
  }, [editable]);

  useEffect(() => {
    if (!editable) return;
    updateScale();
    const observer = new ResizeObserver(updateScale);
    if (containerRef.current) {
      observer.observe(containerRef.current);
    }
    return () => observer.disconnect();
  }, [editable, updateScale]);

  // Measure content height and calculate smart page breaks
  useEffect(() => {
    if (!measureRef.current) return;
    const el = measureRef.current;

    const recalculate = () => {
      const newPages = calculatePageBreaks(el);
      setPages((prev) => (pagesEqual(prev, newPages) ? prev : newPages));
    };

    const observer = new ResizeObserver(recalculate);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  if (!resume) return null;

  const TemplateComponent =
    TEMPLATE_MAP[resume.template_id] ?? ProfessionalTemplate;

  // Font zoom: templates use rem-based Tailwind classes (text-xs etc.) which
  // don't respond to parent font-size. Zoom scales everything proportionally.
  const fontZoom = resume.font_size > 0 ? resume.font_size / 12 : 1;
  // Unitless line-height so children inherit the ratio, not an absolute value.
  const lineHeight = resume.spacing > 0 ? resume.spacing / 100 : 1.15;

  const outerStyle = {
    width: "210mm" as const,
    boxSizing: "border-box" as const,
    padding: `${resume.margin_top}px ${resume.margin_right}px ${resume.margin_bottom}px ${resume.margin_left}px`,
  };

  const contentStyle: React.CSSProperties = {
    fontFamily: resume.font_family,
    lineHeight,
    zoom: fontZoom,
  };

  return (
    <div ref={containerRef} className="flex justify-center p-6">
      <div
        className="origin-top"
        style={{
          transform: `scale(${scale})`,
          transformOrigin: "top center",
        }}
      >
        {/* Hidden measurer — identical styles, measures total content height */}
        <div
          ref={measureRef}
          aria-hidden="true"
          className="text-black"
          style={{
            ...outerStyle,
            position: "absolute",
            visibility: "hidden",
            pointerEvents: "none",
          }}
        >
          <div style={contentStyle}>
            <TemplateComponent editable={false} />
          </div>
        </div>

        {/* Visible page cards */}
        <div className="flex flex-col items-center gap-8">
          {pages.map((page, i) => (
            <div
              key={i}
              className="relative bg-white text-black shadow-lg"
              style={{
                width: "210mm",
                height: A4_HEIGHT_PX,
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  ...outerStyle,
                  marginTop: -page.start,
                  pointerEvents: editable ? "auto" : "none",
                }}
              >
                <div style={contentStyle}>
                  <TemplateComponent editable={editable} />
                </div>
              </div>
              {/* White mask to hide content that belongs to the next page */}
              {page.whiteSpace > 0 && (
                <div
                  className="absolute bottom-0 left-0 right-0 bg-white"
                  style={{ height: page.whiteSpace }}
                />
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
