import { useMemo } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { TEMPLATE_MAP } from "../../lib/templateRegistry";
import { ProfessionalTemplate } from "../preview/ProfessionalTemplate";

/** Width of A4 in px at 96dpi */
const A4_WIDTH = 793;
/** Default thumbnail width for sidebar popover */
const DEFAULT_THUMB_WIDTH = 72;

interface TemplateThumbnailProps {
  readonly templateId: string;
  readonly width?: number;
}

export function TemplateThumbnail({
  templateId,
  width = DEFAULT_THUMB_WIDTH,
}: TemplateThumbnailProps) {
  const resume = useResumeBuilderStore((s) => s.resume);
  const scale = width / A4_WIDTH;

  const TemplateComponent = useMemo(
    () => TEMPLATE_MAP[templateId] ?? ProfessionalTemplate,
    [templateId],
  );

  if (!resume) return null;

  return (
    <div
      className="overflow-hidden rounded border bg-white text-black"
      style={{ width, height: width * 1.414 }}
    >
      <div
        className="pointer-events-none origin-top-left"
        style={{
          transform: `scale(${scale})`,
          width: A4_WIDTH,
          minHeight: A4_WIDTH * 1.414,
          fontFamily: resume.font_family,
          padding: `${resume.margin_top}px ${resume.margin_right}px ${resume.margin_bottom}px ${resume.margin_left}px`,
          lineHeight: `${resume.spacing}%`,
        }}
      >
        <TemplateComponent editable={false} />
      </div>
    </div>
  );
}
