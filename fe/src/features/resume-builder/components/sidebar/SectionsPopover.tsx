import { useTranslation } from "react-i18next";
import { SidebarPopover } from "./SidebarPopover";
import { SectionsConfigurator } from "./SectionsConfigurator";

interface SectionsPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly fullscreen?: boolean;
}

export function SectionsPopover({
  isOpen,
  onClose,
  fullscreen = false,
}: SectionsPopoverProps) {
  const { t } = useTranslation();

  return (
    <SidebarPopover
      isOpen={isOpen}
      onClose={onClose}
      title={t("resumeBuilder.sections.title")}
      fullscreen={fullscreen}
    >
      <SectionsConfigurator />
    </SidebarPopover>
  );
}
