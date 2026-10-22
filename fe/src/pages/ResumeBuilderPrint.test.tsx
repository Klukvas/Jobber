import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { TEMPLATE_MAP } from "@/features/resume-builder/lib/templateRegistry";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import ResumeBuilderPrint from "./ResumeBuilderPrint";

function resume(templateId: string): FullResumeDTO {
  return {
    id: "print-fixture",
    title: "Synthetic resume",
    template_id: templateId,
    font_family: "Georgia",
    primary_color: "#2563eb",
    text_color: "#111111",
    spacing: 115,
    margin_top: 40,
    margin_right: 40,
    margin_bottom: 40,
    margin_left: 40,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    skill_display: "text-only",
    created_at: "",
    updated_at: "",
    contact: {
      full_name: "Synthetic Name",
      email: "",
      phone: "",
      location: "",
      website: "",
      linkedin: "",
      github: "",
    },
    summary: { content: "Summary content marker" },
    experiences: [
      {
        id: "hidden",
        company: "PRIVATE HIDDEN COMPANY",
        position: "",
        location: "",
        start_date: "",
        end_date: "",
        is_current: false,
        description: "",
        sort_order: 0,
      },
    ],
    educations: [
      {
        id: "partial",
        institution: "Partial Education Marker",
        degree: "",
        field_of_study: "",
        start_date: "",
        end_date: "",
        is_current: false,
        gpa: "",
        description: "",
        sort_order: 0,
      },
    ],
    skills: [
      { id: "skill", name: "Skill Content Marker", level: "", sort_order: 0 },
    ],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
    section_order: [
      {
        section_key: "skills",
        sort_order: 0,
        is_visible: true,
        column: "main",
      },
      {
        section_key: "education",
        sort_order: 1,
        is_visible: true,
        column: "main",
      },
      {
        section_key: "summary",
        sort_order: 2,
        is_visible: true,
        column: "main",
      },
      {
        section_key: "experience",
        sort_order: 3,
        is_visible: false,
        column: "main",
      },
    ],
  };
}

afterEach(() => {
  cleanup();
  delete window.__RESUME_DATA__;
  delete window.__PDF_READY__;
  vi.restoreAllMocks();
});

it.each(Object.keys(TEMPLATE_MAP))(
  "prints saved content only, with section visibility and order: %s",
  async (templateId) => {
    const data = resume(templateId);
    const before = JSON.stringify(data);
    window.__RESUME_DATA__ = data;
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      callback(0);
      return 0;
    });
    const { container } = render(<ResumeBuilderPrint />);
    await screen.findByText("Synthetic Name");
    await waitFor(() => expect(window.__PDF_READY__).toBe(true));
    const content = container.textContent ?? "";
    expect(content).toContain("Partial Education Marker");
    for (const placeholder of [
      "Add dates",
      "Position",
      "Degree",
      "Field of Study",
      "Your Name",
      "PRIVATE HIDDEN COMPANY",
    ]) {
      expect(content).not.toContain(placeholder);
    }
    expect(content.indexOf("Skill Content Marker")).toBeLessThan(
      content.indexOf("Partial Education Marker"),
    );
    expect(content.indexOf("Partial Education Marker")).toBeLessThan(
      content.indexOf("Summary content marker"),
    );
    expect(JSON.stringify(data)).toBe(before);
  },
);

it.each(
  Object.keys(TEMPLATE_MAP).flatMap((templateId) => [
    { templateId, blankRows: true },
    { templateId, blankRows: false },
  ]),
)(
  "omits empty headings from read-only output: $templateId blankRows=$blankRows",
  async ({ templateId, blankRows }) => {
    const data = resume(templateId);
    const blankWithRows: FullResumeDTO = {
      ...data,
      contact: {
        full_name: "",
        email: "",
        phone: "",
        location: "",
        website: "",
        linkedin: "",
        github: "",
      },
      summary: { content: " \n " },
      experiences: [{ ...data.experiences[0], company: " " }],
      educations: [{ ...data.educations[0], institution: " " }],
      skills: [{ ...data.skills[0], name: " " }],
      languages: [{ id: "blank", name: "", proficiency: "", sort_order: 0 }],
      certifications: [
        {
          id: "blank",
          name: "",
          issuer: "",
          issue_date: "",
          expiry_date: "",
          url: "",
          sort_order: 0,
        },
      ],
      projects: [
        {
          id: "blank",
          name: "",
          url: "",
          start_date: "",
          end_date: "",
          description: "",
          sort_order: 0,
        },
      ],
      volunteering: [
        {
          id: "blank",
          organization: "",
          role: "",
          start_date: "",
          end_date: "",
          description: "",
          sort_order: 0,
        },
      ],
      custom_sections: [{ id: "blank", title: "", content: "", sort_order: 0 }],
      section_order: [
        "contact",
        "summary",
        "experience",
        "education",
        "skills",
        "languages",
        "certifications",
        "projects",
        "volunteering",
        "custom",
      ].map((section_key, sort_order) => ({
        section_key,
        sort_order,
        is_visible: true,
        column: "main",
      })),
    };
    const blank = blankRows
      ? blankWithRows
      : {
          ...blankWithRows,
          experiences: [],
          educations: [],
          skills: [],
          languages: [],
          certifications: [],
          projects: [],
          volunteering: [],
          custom_sections: [],
        };
    const before = JSON.stringify(blank);
    window.__RESUME_DATA__ = blank;
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      callback(0);
      return 0;
    });
    render(<ResumeBuilderPrint />);
    await waitFor(() => expect(window.__PDF_READY__).toBe(true));
    expect(document.querySelectorAll("h1, h2, h3, h4, h5, h6")).toHaveLength(0);
    expect(JSON.stringify(blank)).toBe(before);
  },
);
