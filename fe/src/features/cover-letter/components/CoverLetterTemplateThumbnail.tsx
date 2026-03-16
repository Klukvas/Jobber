interface CoverLetterTemplateThumbnailProps {
  templateId: string;
  accentColor?: string;
  size?: "sm" | "lg";
}

function ProfessionalThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Horizontal accent line at top */}
      <rect x="6" y="8" width="36" height="2" rx="1" fill={accentColor} />
      {/* Header text lines */}
      <rect x="6" y="14" width="24" height="2" rx="1" fill="#d1d5db" />
      <rect x="6" y="18" width="18" height="2" rx="1" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="26" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="30" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="34" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="38" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="44" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect x="6" y="56" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function ModernThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Vertical accent bar */}
      <rect x="6" y="8" width="3" height="16" rx="1.5" fill={accentColor} />
      {/* Header text lines */}
      <rect
        x="12"
        y="9"
        width="20"
        height="2"
        rx="1"
        fill={accentColor}
        opacity="0.7"
      />
      <rect x="12" y="14" width="16" height="2" rx="1" fill="#d1d5db" />
      <rect x="12" y="19" width="12" height="2" rx="1" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="30" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="34" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="38" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="42" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="52" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect
        x="6"
        y="58"
        width="14"
        height="2"
        rx="1"
        fill={accentColor}
        opacity="0.7"
      />
    </svg>
  );
}

function MinimalThumbnail() {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Simple centered header text */}
      <rect x="10" y="10" width="20" height="2" rx="1" fill="#d1d5db" />
      <rect x="10" y="14" width="14" height="2" rx="1" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="24" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="28" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="32" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="36" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="42" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="46" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="50" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect x="6" y="58" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function ExecutiveThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Bold colored header block */}
      <rect x="4" y="4" width="40" height="16" rx="2" fill={accentColor} />
      <rect x="8" y="8" width="18" height="2" rx="1" fill="white" />
      <rect
        x="8"
        y="12"
        width="14"
        height="2"
        rx="1"
        fill="white"
        opacity="0.7"
      />
      {/* Date */}
      <rect x="6" y="24" width="16" height="1.5" rx="0.75" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="30" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="34" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="38" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="42" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="52" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect x="6" y="58" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function CreativeThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Left colored sidebar */}
      <rect x="4" y="4" width="12" height="56" rx="2" fill={accentColor} />
      <rect x="6" y="8" width="8" height="1.5" rx="0.75" fill="white" />
      <rect
        x="6"
        y="12"
        width="6"
        height="1.5"
        rx="0.75"
        fill="white"
        opacity="0.7"
      />
      <rect
        x="6"
        y="18"
        width="8"
        height="1.5"
        rx="0.75"
        fill="white"
        opacity="0.7"
      />
      <rect
        x="6"
        y="22"
        width="6"
        height="1.5"
        rx="0.75"
        fill="white"
        opacity="0.5"
      />
      {/* Right content area */}
      <rect x="20" y="8" width="22" height="1.5" rx="0.75" fill="#d1d5db" />
      <rect x="20" y="14" width="22" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="18" width="20" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="22" width="22" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="26" width="18" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="32" width="22" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="36" width="20" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="40" width="22" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="20" y="44" width="16" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect x="20" y="52" width="14" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function ClassicThumbnail() {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Horizontal rule above header */}
      <rect x="4" y="7" width="40" height="0.5" fill="#d1d5db" />
      {/* Centered header lines */}
      <rect x="14" y="10" width="20" height="2" rx="1" fill="#d1d5db" />
      <rect x="16" y="14" width="16" height="2" rx="1" fill="#d1d5db" />
      {/* Horizontal rule below header */}
      <rect x="4" y="19" width="40" height="0.5" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="24" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="28" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="32" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="36" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="42" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="46" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing line */}
      <rect x="6" y="56" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function ElegantThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Thin colored stripe at very top */}
      <rect x="0" y="0" width="48" height="3" fill={accentColor} />
      {/* Right-aligned date line */}
      <rect x="28" y="10" width="14" height="1.5" rx="0.75" fill="#d1d5db" />
      {/* Body text lines */}
      <rect x="6" y="18" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="22" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="26" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="30" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="36" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="40" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="44" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="30" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Colored closing line */}
      <rect
        x="6"
        y="56"
        width="16"
        height="2"
        rx="1"
        fill={accentColor}
        opacity="0.7"
      />
    </svg>
  );
}

function BoldThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Large colored band at top */}
      <rect x="0" y="0" width="48" height="20" fill={accentColor} />
      {/* White text line inside band */}
      <rect x="8" y="8" width="20" height="2" rx="1" fill="white" />
      <rect
        x="8"
        y="13"
        width="14"
        height="2"
        rx="1"
        fill="white"
        opacity="0.7"
      />
      {/* Body lines with small colored left border indicators */}
      <rect x="6" y="26" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="26" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="30" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="30" width="30" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="34" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="34" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="38" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="38" width="24" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="44" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="44" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="2" height="1.5" fill={accentColor} />
      <rect x="10" y="48" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
    </svg>
  );
}

function SimpleThumbnail() {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Gray header lines */}
      <rect x="6" y="8" width="22" height="2" rx="1" fill="#d1d5db" />
      <rect x="6" y="12" width="16" height="2" rx="1" fill="#d1d5db" />
      {/* Gray body text lines */}
      <rect x="6" y="22" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="26" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="30" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="34" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="40" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="44" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="48" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Gray closing */}
      <rect x="6" y="56" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

function CorporateThumbnail({ accentColor }: { accentColor: string }) {
  return (
    <svg width="48" height="64" viewBox="0 0 48 64" fill="none">
      <rect width="48" height="64" rx="2" fill="white" />
      {/* Left header group */}
      <rect x="6" y="8" width="16" height="2" rx="1" fill="#d1d5db" />
      <rect x="6" y="12" width="12" height="1.5" rx="0.75" fill="#d1d5db" />
      {/* Right header group */}
      <rect x="28" y="8" width="14" height="2" rx="1" fill="#d1d5db" />
      <rect x="30" y="12" width="12" height="1.5" rx="0.75" fill="#d1d5db" />
      {/* Colored bottom border under header */}
      <rect x="4" y="18" width="40" height="1.5" rx="0.75" fill={accentColor} />
      {/* Body text lines */}
      <rect x="6" y="24" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="28" width="34" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="32" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="36" width="28" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="42" width="36" height="1.5" rx="0.75" fill="#e5e7eb" />
      <rect x="6" y="46" width="32" height="1.5" rx="0.75" fill="#e5e7eb" />
      {/* Closing */}
      <rect x="6" y="56" width="16" height="2" rx="1" fill="#d1d5db" />
    </svg>
  );
}

const THUMBNAIL_MAP: Record<
  string,
  React.ComponentType<{ accentColor: string }>
> = {
  professional: ProfessionalThumbnail,
  modern: ModernThumbnail,
  minimal: () => <MinimalThumbnail />,
  executive: ExecutiveThumbnail,
  creative: CreativeThumbnail,
  classic: () => <ClassicThumbnail />,
  elegant: ElegantThumbnail,
  bold: BoldThumbnail,
  simple: () => <SimpleThumbnail />,
  corporate: CorporateThumbnail,
};

export function CoverLetterTemplateThumbnail({
  templateId,
  accentColor = "#2563eb",
  size = "sm",
}: CoverLetterTemplateThumbnailProps) {
  const Component = THUMBNAIL_MAP[templateId];
  if (!Component) return null;

  if (size === "lg") {
    return (
      <div className="[&>svg]:h-[128px] [&>svg]:w-[96px]">
        <Component accentColor={accentColor} />
      </div>
    );
  }

  return <Component accentColor={accentColor} />;
}
