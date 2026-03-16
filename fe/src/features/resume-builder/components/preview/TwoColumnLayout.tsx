import type { CSSProperties, ReactNode } from "react";

interface TwoColumnLayoutProps {
  readonly sidebarWidth: number;
  readonly layoutMode: "double-left" | "double-right";
  readonly mainContent: ReactNode;
  readonly sidebarContent: ReactNode;
  readonly sidebarStyle?: CSSProperties;
}

export function TwoColumnLayout({
  sidebarWidth,
  layoutMode,
  mainContent,
  sidebarContent,
  sidebarStyle,
}: TwoColumnLayoutProps) {
  const sidebarWidthPercent = `${sidebarWidth}%`;
  const mainWidthPercent = `${100 - sidebarWidth}%`;

  const sidebar = (
    <div style={{ width: sidebarWidthPercent, ...sidebarStyle }}>
      {sidebarContent}
    </div>
  );

  const main = (
    <div style={{ width: mainWidthPercent }}>{mainContent}</div>
  );

  return (
    <div className="flex min-h-full">
      {layoutMode === "double-left" ? (
        <>
          {sidebar}
          {main}
        </>
      ) : (
        <>
          {main}
          {sidebar}
        </>
      )}
    </div>
  );
}
