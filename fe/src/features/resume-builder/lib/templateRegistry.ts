import type React from "react";
import { ProfessionalTemplate } from "../components/preview/ProfessionalTemplate";
import { ModernTemplate } from "../components/preview/ModernTemplate";
import { MinimalTemplate } from "../components/preview/MinimalTemplate";
import { ExecutiveTemplate } from "../components/preview/ExecutiveTemplate";
import { CreativeTemplate } from "../components/preview/CreativeTemplate";
import { CompactTemplate } from "../components/preview/CompactTemplate";
import { ElegantTemplate } from "../components/preview/ElegantTemplate";
import { IconicTemplate } from "../components/preview/IconicTemplate";
import { BoldTemplate } from "../components/preview/BoldTemplate";
import { AccentTemplate } from "../components/preview/AccentTemplate";
import { TimelineTemplate } from "../components/preview/TimelineTemplate";
import { VividTemplate } from "../components/preview/VividTemplate";

/** Maps template UUIDs to their React preview components. Single source of truth. */
export const TEMPLATE_MAP: Record<
  string,
  React.ComponentType<{ editable?: boolean }>
> = {
  "00000000-0000-0000-0000-000000000001": ProfessionalTemplate,
  "00000000-0000-0000-0000-000000000002": ModernTemplate,
  "00000000-0000-0000-0000-000000000003": MinimalTemplate,
  "00000000-0000-0000-0000-000000000004": ExecutiveTemplate,
  "00000000-0000-0000-0000-000000000005": CreativeTemplate,
  "00000000-0000-0000-0000-000000000006": CompactTemplate,
  "00000000-0000-0000-0000-000000000007": ElegantTemplate,
  "00000000-0000-0000-0000-000000000008": IconicTemplate,
  "00000000-0000-0000-0000-000000000009": BoldTemplate,
  "00000000-0000-0000-0000-00000000000a": AccentTemplate,
  "00000000-0000-0000-0000-00000000000b": TimelineTemplate,
  "00000000-0000-0000-0000-00000000000c": VividTemplate,
};

/** Template metadata for the picker UI. */
export const TEMPLATE_LIST = [
  {
    id: "00000000-0000-0000-0000-000000000001",
    nameKey: "resumeBuilder.templates.professional",
  },
  {
    id: "00000000-0000-0000-0000-000000000002",
    nameKey: "resumeBuilder.templates.modern",
  },
  {
    id: "00000000-0000-0000-0000-000000000003",
    nameKey: "resumeBuilder.templates.minimal",
  },
  {
    id: "00000000-0000-0000-0000-000000000004",
    nameKey: "resumeBuilder.templates.executive",
  },
  {
    id: "00000000-0000-0000-0000-000000000005",
    nameKey: "resumeBuilder.templates.creative",
  },
  {
    id: "00000000-0000-0000-0000-000000000006",
    nameKey: "resumeBuilder.templates.compact",
  },
  {
    id: "00000000-0000-0000-0000-000000000007",
    nameKey: "resumeBuilder.templates.elegant",
  },
  {
    id: "00000000-0000-0000-0000-000000000008",
    nameKey: "resumeBuilder.templates.iconic",
  },
  {
    id: "00000000-0000-0000-0000-000000000009",
    nameKey: "resumeBuilder.templates.bold",
  },
  {
    id: "00000000-0000-0000-0000-00000000000a",
    nameKey: "resumeBuilder.templates.accent",
  },
  {
    id: "00000000-0000-0000-0000-00000000000b",
    nameKey: "resumeBuilder.templates.timeline",
  },
  {
    id: "00000000-0000-0000-0000-00000000000c",
    nameKey: "resumeBuilder.templates.vivid",
  },
] as const;
