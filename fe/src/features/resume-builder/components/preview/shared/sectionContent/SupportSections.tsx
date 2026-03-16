import { EditableField } from "../../../inline/EditableField";
import { EditableTextarea } from "../../../inline/EditableTextarea";
import { EntryWrapper } from "../../../inline/EntryWrapper";
import type { SectionContentProps } from "./types";

// ---------------------------------------------------------------------------
// Certifications
// ---------------------------------------------------------------------------

export function CertificationsContent({
  setup,
  config,
  editable,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateCertification, certificationsSection } = setup;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.certification ?? "mb-1";
  const isModern = config.variant === "modern";
  const isMinimal = config.variant === "minimal";

  return (
    <>
      {resume.certifications.map((cert) => (
        <EntryWrapper
          key={cert.id}
          entryId={cert.id}
          onRemove={certificationsSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          {isModern ? (
            <>
              <EditableField
                value={cert.name}
                onChange={(v) => updateCertification(cert.id, { name: v })}
                placeholder="Certification"
                as="p"
                className="text-xs font-medium"
                inputClassName={inputClass}
                editable={editable}
              />
              {(cert.issuer || editable) && (
                <EditableField
                  value={cert.issuer}
                  onChange={(v) =>
                    updateCertification(cert.id, { issuer: v })
                  }
                  placeholder="Issuer"
                  as="p"
                  className="text-xs opacity-70"
                  inputClassName={inputClass}
                  editable={editable}
                />
              )}
            </>
          ) : isMinimal ? (
            <p className="text-xs text-gray-600">
              <EditableField
                value={cert.name}
                onChange={(v) => updateCertification(cert.id, { name: v })}
                placeholder="Certification"
                inputClassName={inputClass}
                editable={editable}
              />
              {(cert.issuer || editable) && (
                <>
                  {cert.name && cert.issuer ? " \u2014 " : ""}
                  <EditableField
                    value={cert.issuer}
                    onChange={(v) =>
                      updateCertification(cert.id, { issuer: v })
                    }
                    placeholder="Issuer"
                    inputClassName={inputClass}
                    editable={editable}
                  />
                </>
              )}
            </p>
          ) : (
            <>
              <div
                className={`flex items-center gap-1 ${config.textSize} font-bold`}
              >
                <EditableField
                  value={cert.name}
                  onChange={(v) => updateCertification(cert.id, { name: v })}
                  placeholder="Certification name"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(cert.issuer || editable) && (
                  <>
                    <span>{cert.name && cert.issuer ? " \u2014 " : ""}</span>
                    <EditableField
                      value={cert.issuer}
                      onChange={(v) =>
                        updateCertification(cert.id, { issuer: v })
                      }
                      placeholder="Issuer"
                      className="font-normal"
                      inputClassName={inputClass}
                      editable={editable}
                    />
                  </>
                )}
              </div>
              {(cert.issue_date || editable) && (
                <EditableField
                  value={cert.issue_date}
                  onChange={(v) =>
                    updateCertification(cert.id, { issue_date: v })
                  }
                  placeholder="Issue date"
                  type="date"
                  as="p"
                  className={`${config.textSize} text-gray-500`}
                  editable={editable}
                />
              )}
            </>
          )}
        </EntryWrapper>
      ))}
    </>
  );
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

export function ProjectsContent({
  setup,
  config,
  editable,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateProject, projectsSection } = setup;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.project ?? "mb-2";
  const isMinimal = config.variant === "minimal";

  return (
    <>
      {resume.projects.map((proj) => (
        <EntryWrapper
          key={proj.id}
          entryId={proj.id}
          onRemove={projectsSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          <EditableField
            value={proj.name}
            onChange={(v) => updateProject(proj.id, { name: v })}
            placeholder="Project name"
            as="p"
            className={`${config.textSize} ${isMinimal ? "font-semibold" : "font-bold"}`}
            inputClassName={inputClass}
            editable={editable}
          />
          {(proj.description || editable) && (
            <EditableTextarea
              value={proj.description}
              onChange={(v) => updateProject(proj.id, { description: v })}
              placeholder="Project description..."
              className={`${config.textSize} ${config.variant === "compact" ? "leading-snug" : ""} text-gray-700`}
              editable={editable}
            />
          )}
        </EntryWrapper>
      ))}
    </>
  );
}

// ---------------------------------------------------------------------------
// Volunteering
// ---------------------------------------------------------------------------

export function VolunteeringContent({
  setup,
  config,
  editable,
  sectionColor,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateVolunteering, volunteeringSection } = setup;
  const effectiveColor = sectionColor ?? setup.color;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.volunteering ?? "mb-2";
  const isMinimal = config.variant === "minimal";
  const isModern = config.variant === "modern";

  return (
    <>
      {resume.volunteering.map((vol) => (
        <EntryWrapper
          key={vol.id}
          entryId={vol.id}
          onRemove={volunteeringSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          {isModern ? (
            <>
              <EditableField
                value={vol.role}
                onChange={(v) => updateVolunteering(vol.id, { role: v })}
                placeholder="Role"
                as="p"
                className="text-xs font-bold"
                inputClassName={inputClass}
                editable={editable}
              />
              <EditableField
                value={vol.organization}
                onChange={(v) =>
                  updateVolunteering(vol.id, { organization: v })
                }
                placeholder="Organization"
                as="p"
                className="text-xs"
                style={{ color: effectiveColor }}
                inputClassName={inputClass}
                editable={editable}
              />
            </>
          ) : (
            <>
              <div
                className={`flex items-center gap-1 ${config.textSize} font-bold`}
              >
                <EditableField
                  value={vol.role}
                  onChange={(v) => updateVolunteering(vol.id, { role: v })}
                  placeholder="Role"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(vol.organization || editable) && (
                  <>
                    <span>
                      {vol.role && vol.organization ? " at " : ""}
                    </span>
                    <EditableField
                      value={vol.organization}
                      onChange={(v) =>
                        updateVolunteering(vol.id, { organization: v })
                      }
                      placeholder="Organization"
                      className={isMinimal ? "font-normal text-gray-500" : "font-normal"}
                      inputClassName={inputClass}
                      editable={editable}
                    />
                  </>
                )}
              </div>
              {(vol.description || editable) && (
                <EditableTextarea
                  value={vol.description}
                  onChange={(v) =>
                    updateVolunteering(vol.id, { description: v })
                  }
                  placeholder="Describe your volunteering..."
                  className={`${config.textSize} ${config.variant === "compact" ? "leading-snug" : ""} text-gray-700`}
                  editable={editable}
                />
              )}
            </>
          )}
        </EntryWrapper>
      ))}
    </>
  );
}

// ---------------------------------------------------------------------------
// Custom Sections
// ---------------------------------------------------------------------------

export function CustomSectionsContent({
  setup,
  config,
  editable,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateCustomSection, customSectionsSection } = setup;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.customSection ?? "mb-2";
  const isMinimal = config.variant === "minimal";

  return (
    <>
      {resume.custom_sections.map((cs) => (
        <EntryWrapper
          key={cs.id}
          entryId={cs.id}
          onRemove={customSectionsSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          <EditableField
            value={cs.title}
            onChange={(v) => updateCustomSection(cs.id, { title: v })}
            placeholder="Section title"
            as="p"
            className={`${config.textSize} ${isMinimal ? "font-medium" : "font-bold"}`}
            inputClassName={inputClass}
            editable={editable}
          />
          <EditableTextarea
            value={cs.content}
            onChange={(v) => updateCustomSection(cs.id, { content: v })}
            placeholder="Section content..."
            className={`${config.textSize} ${config.variant === "compact" ? "leading-snug" : ""} ${isMinimal ? "text-gray-600" : "text-gray-700"}`}
            editable={editable}
          />
        </EntryWrapper>
      ))}
    </>
  );
}
