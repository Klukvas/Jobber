import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("../../../inline/EditableField", () => ({
  EditableField: ({
    value,
    placeholder,
    className,
    style,
  }: {
    value: string;
    placeholder: string;
    className?: string;
    style?: React.CSSProperties;
  }) => (
    <span
      data-testid={`field-${placeholder}`}
      className={className}
      style={style}
    >
      {value}
    </span>
  ),
}));
vi.mock("../../../inline/EditableTextarea", () => ({
  EditableTextarea: ({
    value,
    className,
  }: {
    value: string;
    className?: string;
  }) => (
    <span data-testid="textarea" className={className}>
      {value}
    </span>
  ),
}));
vi.mock("../../../inline/EntryWrapper", () => ({
  EntryWrapper: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => (
    <div data-testid="entry-wrapper" className={className}>
      {children}
    </div>
  ),
}));

import {
  CertificationsContent,
  ProjectsContent,
  VolunteeringContent,
  CustomSectionsContent,
} from "./SupportSections";
import type { TemplateConfig } from "../templateConfig";

function makeConfig(
  variant: TemplateConfig["variant"],
  overrides?: Partial<TemplateConfig>,
): TemplateConfig {
  return {
    variant,
    summaryTitle: "Summary",
    textSize: "text-xs",
    leadingClass: "leading-relaxed",
    skills: { renderAs: "pill", containerClass: "" },
    languages: { renderAs: "flex", containerClass: "" },
    ...overrides,
  };
}

// ============================================================================
// Certifications
// ============================================================================

describe("CertificationsContent", () => {
  function makeSetup(certs: Array<Record<string, unknown>> = []) {
    return {
      resume: {
        certifications: certs,
        primary_color: "#e11d48",
        section_order: [],
      },
      color: "#e11d48",
      updateCertification: vi.fn(),
      certificationsSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    } as unknown as Parameters<typeof CertificationsContent>[0]["setup"];
  }

  const sampleCert = {
    id: "c1",
    name: "AWS Solutions Architect",
    issuer: "Amazon",
    issue_date: "2023-01",
  };

  describe("modern variant", () => {
    it("renders name and issuer with opacity-70", () => {
      const setup = makeSetup([sampleCert]);
      render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Certification")).toHaveTextContent(
        "AWS Solutions Architect",
      );
      const issuerField = screen.getByTestId("field-Issuer");
      expect(issuerField.className).toContain("opacity-70");
    });

    it("hides issuer when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleCert, issuer: "" }]);
      render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("field-Issuer")).not.toBeInTheDocument();
    });
  });

  describe("minimal variant", () => {
    it("renders name and issuer with em-dash separator", () => {
      const setup = makeSetup([sampleCert]);
      render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Certification")).toHaveTextContent(
        "AWS Solutions Architect",
      );
      expect(screen.getByTestId("field-Issuer")).toHaveTextContent("Amazon");
      expect(screen.getByText(/\u2014/)).toBeInTheDocument();
    });
  });

  describe("default variant", () => {
    it("renders name as bold with issuer and issue_date", () => {
      const setup = makeSetup([sampleCert]);
      const { container } = render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      const nameRow = container.querySelector(".font-bold");
      expect(nameRow).toBeTruthy();
      expect(screen.getByTestId("field-Certification name")).toHaveTextContent(
        "AWS Solutions Architect",
      );
      expect(screen.getByTestId("field-Issue date")).toHaveTextContent(
        "2023-01",
      );
    });

    it("hides issue_date when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleCert, issue_date: "" }]);
      render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("field-Issue date")).not.toBeInTheDocument();
    });

    it("shows issue_date when empty but editable", () => {
      const setup = makeSetup([{ ...sampleCert, issue_date: "" }]);
      render(
        <CertificationsContent
          setup={setup}
          config={makeConfig("professional")}
          editable={true}
        />,
      );
      expect(screen.getByTestId("field-Issue date")).toBeInTheDocument();
    });
  });
});

// ============================================================================
// Projects
// ============================================================================

describe("ProjectsContent", () => {
  function makeSetup(projects: Array<Record<string, unknown>> = []) {
    return {
      resume: { projects, primary_color: "#e11d48", section_order: [] },
      color: "#e11d48",
      updateProject: vi.fn(),
      projectsSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    } as unknown as Parameters<typeof ProjectsContent>[0]["setup"];
  }

  const sampleProject = {
    id: "p1",
    name: "Open Source CLI",
    description: "A powerful tool",
  };

  it("renders project name with font-bold for non-minimal", () => {
    const setup = makeSetup([sampleProject]);
    render(
      <ProjectsContent
        setup={setup}
        config={makeConfig("professional")}
        editable={false}
      />,
    );
    const name = screen.getByTestId("field-Project name");
    expect(name.className).toContain("font-bold");
  });

  it("renders project name with font-semibold for minimal", () => {
    const setup = makeSetup([sampleProject]);
    render(
      <ProjectsContent
        setup={setup}
        config={makeConfig("minimal")}
        editable={false}
      />,
    );
    const name = screen.getByTestId("field-Project name");
    expect(name.className).toContain("font-semibold");
  });

  it("renders description when present", () => {
    const setup = makeSetup([sampleProject]);
    render(
      <ProjectsContent
        setup={setup}
        config={makeConfig("professional")}
        editable={false}
      />,
    );
    expect(screen.getByTestId("textarea")).toHaveTextContent("A powerful tool");
  });

  it("hides description when empty and not editable", () => {
    const setup = makeSetup([{ ...sampleProject, description: "" }]);
    render(
      <ProjectsContent
        setup={setup}
        config={makeConfig("professional")}
        editable={false}
      />,
    );
    expect(screen.queryByTestId("textarea")).not.toBeInTheDocument();
  });
});

// ============================================================================
// Volunteering
// ============================================================================

describe("VolunteeringContent", () => {
  function makeSetup(volunteering: Array<Record<string, unknown>> = []) {
    return {
      resume: { volunteering, primary_color: "#e11d48", section_order: [] },
      color: "#e11d48",
      updateVolunteering: vi.fn(),
      volunteeringSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    } as unknown as Parameters<typeof VolunteeringContent>[0]["setup"];
  }

  const sampleVol = {
    id: "v1",
    role: "Mentor",
    organization: "Code.org",
    description: "Teaching kids",
  };

  describe("modern variant", () => {
    it("renders role as bold and organization with effectiveColor", () => {
      const setup = makeSetup([sampleVol]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      const role = screen.getByTestId("field-Role");
      expect(role.className).toContain("font-bold");
      const org = screen.getByTestId("field-Organization");
      expect(org.style.color).toBe("rgb(225, 29, 72)");
    });

    it("uses sectionColor for organization when provided", () => {
      const setup = makeSetup([sampleVol]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
          sectionColor="#3b82f6"
        />,
      );
      const org = screen.getByTestId("field-Organization");
      expect(org.style.color).toBe("rgb(59, 130, 246)");
    });
  });

  describe("default variant", () => {
    it("renders role and organization with 'at' separator", () => {
      const setup = makeSetup([sampleVol]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Role")).toHaveTextContent("Mentor");
      expect(screen.getByTestId("field-Organization")).toHaveTextContent(
        "Code.org",
      );
      expect(screen.getByText(/at/)).toBeInTheDocument();
    });

    it("hides organization when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleVol, organization: "" }]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(
        screen.queryByTestId("field-Organization"),
      ).not.toBeInTheDocument();
    });

    it("renders description", () => {
      const setup = makeSetup([sampleVol]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("textarea")).toHaveTextContent("Teaching kids");
    });

    it("hides description when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleVol, description: "" }]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("textarea")).not.toBeInTheDocument();
    });
  });

  describe("minimal variant", () => {
    it("applies font-normal text-gray-500 to organization", () => {
      const setup = makeSetup([sampleVol]);
      render(
        <VolunteeringContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      const org = screen.getByTestId("field-Organization");
      expect(org.className).toContain("font-normal");
      expect(org.className).toContain("text-gray-500");
    });
  });
});

// ============================================================================
// Custom Sections
// ============================================================================

describe("CustomSectionsContent", () => {
  function makeSetup(customSections: Array<Record<string, unknown>> = []) {
    return {
      resume: {
        custom_sections: customSections,
        primary_color: "#e11d48",
        section_order: [],
      },
      color: "#e11d48",
      updateCustomSection: vi.fn(),
      customSectionsSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    } as unknown as Parameters<typeof CustomSectionsContent>[0]["setup"];
  }

  const sampleCS = {
    id: "cs1",
    title: "Awards",
    content: "Best Developer 2023",
  };

  it("renders title with font-bold for non-minimal", () => {
    const setup = makeSetup([sampleCS]);
    render(
      <CustomSectionsContent
        setup={setup}
        config={makeConfig("professional")}
        editable={false}
      />,
    );
    const title = screen.getByTestId("field-Section title");
    expect(title.className).toContain("font-bold");
  });

  it("renders title with font-medium for minimal", () => {
    const setup = makeSetup([sampleCS]);
    render(
      <CustomSectionsContent
        setup={setup}
        config={makeConfig("minimal")}
        editable={false}
      />,
    );
    const title = screen.getByTestId("field-Section title");
    expect(title.className).toContain("font-medium");
  });

  it("renders content with text-gray-600 for minimal", () => {
    const setup = makeSetup([sampleCS]);
    render(
      <CustomSectionsContent
        setup={setup}
        config={makeConfig("minimal")}
        editable={false}
      />,
    );
    const content = screen.getByTestId("textarea");
    expect(content.className).toContain("text-gray-600");
  });

  it("renders content with text-gray-700 for non-minimal", () => {
    const setup = makeSetup([sampleCS]);
    render(
      <CustomSectionsContent
        setup={setup}
        config={makeConfig("professional")}
        editable={false}
      />,
    );
    const content = screen.getByTestId("textarea");
    expect(content.className).toContain("text-gray-700");
  });
});
