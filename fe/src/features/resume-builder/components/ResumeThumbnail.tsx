import { useRef, useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { TEMPLATE_MAP } from "../lib/templateRegistry";
import { ProfessionalTemplate } from "./preview/ProfessionalTemplate";
import { ResumePreviewProvider } from "./preview/ResumePreviewContext";

const A4_WIDTH_PX = 793;

interface ResumeThumbnailProps {
  readonly resumeId: string;
  readonly templateId: string;
}

export function ResumeThumbnail({
  resumeId,
  templateId,
}: ResumeThumbnailProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(0.25);

  const { data } = useQuery({
    queryKey: ["resume-builder", resumeId],
    queryFn: () => resumeBuilderService.getById(resumeId),
    staleTime: 5 * 60 * 1000,
  });

  useEffect(() => {
    if (!containerRef.current) return;

    const observer = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? 300;
      setScale(width / A4_WIDTH_PX);
    });
    observer.observe(containerRef.current);

    return () => observer.disconnect();
  }, []);

  const TemplateComponent = TEMPLATE_MAP[templateId] ?? ProfessionalTemplate;

  if (!data) {
    return (
      <div
        ref={containerRef}
        className="h-36 animate-pulse rounded-md bg-muted"
      />
    );
  }

  return (
    <div
      ref={containerRef}
      className="relative h-36 overflow-hidden rounded-md border bg-white text-black"
    >
      <ResumePreviewProvider value={data}>
        <div
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            width: A4_WIDTH_PX,
            transformOrigin: "top left",
            transform: `scale(${scale})`,
            fontFamily: data.font_family,
            padding: `${data.margin_top}px ${data.margin_right}px ${data.margin_bottom}px ${data.margin_left}px`,
            lineHeight: `${data.spacing}%`,
            pointerEvents: "none",
          }}
        >
          <TemplateComponent editable={false} />
        </div>
      </ResumePreviewProvider>
    </div>
  );
}
