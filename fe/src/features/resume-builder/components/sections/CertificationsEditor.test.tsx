import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CertificationsEditor } from "./CertificationsEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { CertificationDTO } from "@/shared/types/resume-builder";

const mockState = createMockStoreState();
const mockPersistAdd = vi.fn();
const mockPersistRemove = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("../../hooks/useSectionPersistence", () => ({
  useSectionPersistence: () => ({
    add: mockPersistAdd,
    remove: mockPersistRemove,
  }),
}));

const sampleCertification: CertificationDTO = {
  id: "cert-1",
  name: "AWS Solutions Architect",
  issuer: "Amazon",
  issue_date: "2023-01-15",
  expiry_date: "2026-01-15",
  url: "https://aws.amazon.com/certification",
  sort_order: 0,
};

describe("CertificationsEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<CertificationsEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.certifications"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no certifications", () => {
    render(<CertificationsEditor />);
    expect(
      screen.getByText("resumeBuilder.certifications.empty"),
    ).toBeInTheDocument();
  });

  it("adds certification when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<CertificationsEditor />);

    await user.click(screen.getByText("resumeBuilder.actions.add"));
    expect(mockState.addCertification).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders certification card with name and issuer", () => {
    const stateWithCerts = createMockStoreState({
      certifications: [sampleCertification],
    });
    Object.assign(mockState, stateWithCerts);

    render(<CertificationsEditor />);
    expect(
      screen.getByText("AWS Solutions Architect - Amazon"),
    ).toBeInTheDocument();
  });

  it("displays all certification fields when expanded", () => {
    const stateWithCerts = createMockStoreState({
      certifications: [sampleCertification],
    });
    Object.assign(mockState, stateWithCerts);

    render(<CertificationsEditor />);
    expect(
      screen.getByDisplayValue("AWS Solutions Architect"),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("Amazon")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("https://aws.amazon.com/certification"),
    ).toBeInTheDocument();
  });

  it("calls removeCertification when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithCerts = createMockStoreState({
      certifications: [sampleCertification],
    });
    Object.assign(mockState, stateWithCerts);

    render(<CertificationsEditor />);
    await user.click(screen.getByText("resumeBuilder.actions.remove"));

    expect(mockState.removeCertification).toHaveBeenCalledWith("cert-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("cert-1");
  });

  it("toggles card open/closed", async () => {
    const user = userEvent.setup();
    const stateWithCerts = createMockStoreState({
      certifications: [sampleCertification],
    });
    Object.assign(mockState, stateWithCerts);

    render(<CertificationsEditor />);
    await user.click(
      screen.getByText("AWS Solutions Architect - Amazon"),
    );

    expect(
      screen.queryByDisplayValue("AWS Solutions Architect"),
    ).not.toBeInTheDocument();
  });

  it("shows fallback title for empty certification", () => {
    const emptyCert: CertificationDTO = {
      ...sampleCertification,
      name: "",
      issuer: "",
    };
    const stateWithEmpty = createMockStoreState({
      certifications: [emptyCert],
    });
    Object.assign(mockState, stateWithEmpty);

    render(<CertificationsEditor />);
    expect(
      screen.getByText("resumeBuilder.certifications.newEntry"),
    ).toBeInTheDocument();
  });
});
