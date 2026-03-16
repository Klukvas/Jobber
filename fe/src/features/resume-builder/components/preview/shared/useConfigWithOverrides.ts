import { useMemo } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { TemplateConfig, SkillRenderMode } from "./templateConfig";

/** Container class overrides for each skill display mode. */
const CONTAINER_CLASS_MAP: Partial<Record<SkillRenderMode, string>> = {
  dots: "space-y-1.5",
  bar: "space-y-1.5",
  square: "space-y-1.5",
  star: "space-y-1.5",
  circle: "grid grid-cols-3 gap-4",
  segmented: "space-y-1.5",
  bubble: "flex flex-wrap gap-2",
};

/**
 * Returns a config with the user's skill_display override applied.
 * If skill_display is empty, the original template config is returned unchanged.
 */
export function useConfigWithOverrides(config: TemplateConfig): TemplateConfig {
  const skillDisplay = useResumeBuilderStore(
    (s) => s.resume?.skill_display ?? "",
  );

  return useMemo(() => {
    if (!skillDisplay) return config;

    const renderAs = skillDisplay as SkillRenderMode;
    const containerClass =
      CONTAINER_CLASS_MAP[renderAs] ?? config.skills.containerClass;

    return {
      ...config,
      skills: { renderAs, containerClass },
    };
  }, [config, skillDisplay]);
}
