import { EditableField } from "../../../inline/EditableField";
import { EditableSelect } from "../../../inline/EditableSelect";
import { EntryWrapper } from "../../../inline/EntryWrapper";
import { SKILL_LEVEL_OPTIONS } from "../../../inline/constants";
import {
  skillLevelToNumber,
  MAX_DOTS,
  MAX_SKILL_LEVEL,
} from "./skillLevelUtils";
import type { SectionContentProps } from "./types";

export function SkillsContent({
  setup,
  config,
  editable,
  sectionColor,
  inputClassName,
}: SectionContentProps) {
  const { resume, color, updateSkill, skillsSection } = setup;
  const effectiveColor = sectionColor ?? color;
  const inputClass = inputClassName ?? "";

  switch (config.skills.renderAs) {
    case "text-level":
      return (
        <div className={config.skills.containerClass}>
          {resume.skills.map((skill) => (
            <EntryWrapper
              key={skill.id}
              entryId={skill.id}
              onRemove={skillsSection.handleRemove}
              editable={editable}
            >
              <span className={`${config.textSize} text-gray-700`}>
                <EditableField
                  value={skill.name}
                  onChange={(v) => updateSkill(skill.id, { name: v })}
                  placeholder="Skill name"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(skill.level || editable) && (
                  <>
                    {" ("}
                    <EditableSelect
                      value={skill.level}
                      onChange={(v) => updateSkill(skill.id, { level: v })}
                      options={SKILL_LEVEL_OPTIONS}
                      placeholder="Level"
                      className={config.textSize}
                      editable={editable}
                    />
                    {")"}
                  </>
                )}
              </span>
            </EntryWrapper>
          ))}
        </div>
      );

    case "pill":
      return (
        <div className={config.skills.containerClass}>
          {resume.skills.map((skill) => (
            <EntryWrapper
              key={skill.id}
              entryId={skill.id}
              onRemove={skillsSection.handleRemove}
              editable={editable}
            >
              <span
                className="inline-block rounded-full px-2.5 py-0.5 text-xs font-medium text-white"
                style={{ backgroundColor: effectiveColor }}
              >
                <EditableField
                  value={skill.name}
                  onChange={(v) => updateSkill(skill.id, { name: v })}
                  placeholder="Skill"
                  inputClassName="text-white placeholder:text-white/60"
                  editable={editable}
                />
              </span>
            </EntryWrapper>
          ))}
        </div>
      );

    case "grid-level":
      return (
        <div className={config.skills.containerClass}>
          {resume.skills.map((skill) => (
            <EntryWrapper
              key={skill.id}
              entryId={skill.id}
              onRemove={skillsSection.handleRemove}
              editable={editable}
            >
              <span className={`${config.textSize} text-gray-700`}>
                <EditableField
                  value={skill.name}
                  onChange={(v) => updateSkill(skill.id, { name: v })}
                  placeholder="Skill"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(skill.level || editable) && (
                  <>
                    {" ("}
                    <EditableSelect
                      value={skill.level}
                      onChange={(v) => updateSkill(skill.id, { level: v })}
                      options={SKILL_LEVEL_OPTIONS}
                      placeholder="Level"
                      className={config.textSize}
                      editable={editable}
                    />
                    {")"}
                  </>
                )}
              </span>
            </EntryWrapper>
          ))}
        </div>
      );

    case "vertical":
      return (
        <div className={config.skills.containerClass}>
          {resume.skills.map((skill) => (
            <EntryWrapper
              key={skill.id}
              entryId={skill.id}
              onRemove={skillsSection.handleRemove}
              editable={editable}
            >
              <div>
                <EditableField
                  value={skill.name}
                  onChange={(v) => updateSkill(skill.id, { name: v })}
                  placeholder="Skill name"
                  as="p"
                  className="text-xs font-medium"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(skill.level || editable) && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-xs opacity-70"
                    editable={editable}
                  />
                )}
              </div>
            </EntryWrapper>
          ))}
        </div>
      );

    case "dots":
      return (
        <div className="space-y-1.5">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex items-center justify-between gap-2">
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill name"
                    className={`${config.textSize} font-medium text-gray-700`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  <div className="flex items-center gap-1">
                    {Array.from({ length: MAX_DOTS }, (_, i) => (
                      <span
                        key={i}
                        className={`inline-block h-2.5 w-2.5 rounded-full ${i >= level ? "bg-gray-200" : ""}`}
                        style={
                          i < level
                            ? { backgroundColor: effectiveColor }
                            : undefined
                        }
                      />
                    ))}
                  </div>
                </div>
                {editable && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-[10px] opacity-60"
                    editable={editable}
                  />
                )}
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "bar":
      return (
        <div className="space-y-1.5">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            const widthPercent = (level / MAX_SKILL_LEVEL) * 100;
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex items-center gap-3">
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill name"
                    className={`${config.textSize} min-w-[80px] font-medium text-gray-700`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  <div className="h-2 flex-1 rounded-full bg-gray-200">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${widthPercent}%`,
                        backgroundColor: effectiveColor,
                      }}
                    />
                  </div>
                </div>
                {editable && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-[10px] opacity-60"
                    editable={editable}
                  />
                )}
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "text-only":
      if (editable) {
        return (
          <div className={config.skills.containerClass}>
            {resume.skills.map((s) => (
              <EntryWrapper
                key={s.id}
                entryId={s.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <span className="text-xs text-gray-600">
                  <EditableField
                    value={s.name}
                    onChange={(v) => updateSkill(s.id, { name: v })}
                    placeholder="Skill"
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  {" \u00B7 "}
                </span>
              </EntryWrapper>
            ))}
          </div>
        );
      }
      return resume.skills.length > 0 ? (
        <p className="text-xs text-gray-600">
          {resume.skills.map((s) => s.name).join(" \u00B7 ")}
        </p>
      ) : null;

    case "square":
      return (
        <div className="space-y-1.5">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex items-center justify-between gap-2">
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill name"
                    className={`${config.textSize} font-medium text-gray-700`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  <div className="flex items-center gap-0.5">
                    {Array.from({ length: MAX_DOTS }, (_, i) => (
                      <span
                        key={i}
                        className={`inline-block h-2.5 w-2.5 rounded-sm ${i >= level ? "bg-gray-200" : ""}`}
                        style={
                          i < level
                            ? { backgroundColor: effectiveColor }
                            : undefined
                        }
                      />
                    ))}
                  </div>
                </div>
                {editable && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-[10px] opacity-60"
                    editable={editable}
                  />
                )}
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "star":
      return (
        <div className="space-y-1.5">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex items-center justify-between gap-2">
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill name"
                    className={`${config.textSize} font-medium text-gray-700`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  <div className="flex items-center gap-0.5">
                    {Array.from({ length: MAX_DOTS }, (_, i) => (
                      <span
                        key={i}
                        className="text-sm leading-none"
                        style={{
                          color: i < level ? effectiveColor : "#d1d5db",
                        }}
                      >
                        {"\u2605"}
                      </span>
                    ))}
                  </div>
                </div>
                {editable && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-[10px] opacity-60"
                    editable={editable}
                  />
                )}
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "circle":
      return (
        <div className="grid grid-cols-3 gap-4">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            const pct = (level / MAX_SKILL_LEVEL) * 100;
            const radius = 14;
            const circumference = 2 * Math.PI * radius;
            const dashOffset = circumference - (circumference * pct) / 100;
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex flex-col items-center gap-1">
                  <svg width="36" height="36" viewBox="0 0 36 36">
                    <circle
                      cx="18"
                      cy="18"
                      r={radius}
                      fill="none"
                      stroke="#e5e7eb"
                      strokeWidth="3"
                    />
                    <circle
                      cx="18"
                      cy="18"
                      r={radius}
                      fill="none"
                      stroke={effectiveColor}
                      strokeWidth="3"
                      strokeDasharray={circumference}
                      strokeDashoffset={dashOffset}
                      strokeLinecap="round"
                      transform="rotate(-90 18 18)"
                    />
                    <text
                      x="18"
                      y="18"
                      textAnchor="middle"
                      dominantBaseline="central"
                      className="text-[8px] font-bold fill-gray-600"
                    >
                      {pct > 0 ? `${pct}%` : "–"}
                    </text>
                  </svg>
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill"
                    className="text-center text-[10px] font-medium text-gray-700"
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  {editable && (
                    <EditableSelect
                      value={skill.level}
                      onChange={(v) => updateSkill(skill.id, { level: v })}
                      options={SKILL_LEVEL_OPTIONS}
                      placeholder="Level"
                      className="text-[10px] opacity-60"
                      editable={editable}
                    />
                  )}
                </div>
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "segmented":
      return (
        <div className="space-y-1.5">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <div className="flex items-center gap-3">
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill name"
                    className={`${config.textSize} min-w-[80px] font-medium text-gray-700`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  <div className="flex flex-1 gap-1">
                    {Array.from({ length: MAX_SKILL_LEVEL }, (_, i) => (
                      <div
                        key={i}
                        className={`h-2 flex-1 rounded-sm ${i >= level ? "bg-gray-200" : ""}`}
                        style={
                          i < level
                            ? { backgroundColor: effectiveColor }
                            : undefined
                        }
                      />
                    ))}
                  </div>
                </div>
                {editable && (
                  <EditableSelect
                    value={skill.level}
                    onChange={(v) => updateSkill(skill.id, { level: v })}
                    options={SKILL_LEVEL_OPTIONS}
                    placeholder="Level"
                    className="text-[10px] opacity-60"
                    editable={editable}
                  />
                )}
              </EntryWrapper>
            );
          })}
        </div>
      );

    case "bubble":
      return (
        <div className="flex flex-wrap gap-2">
          {resume.skills.map((skill) => {
            const level = skillLevelToNumber(skill.level);
            const opacity =
              level > 0 ? 0.15 + (level / MAX_SKILL_LEVEL) * 0.25 : 0.1;
            return (
              <EntryWrapper
                key={skill.id}
                entryId={skill.id}
                onRemove={skillsSection.handleRemove}
                editable={editable}
              >
                <span
                  className="inline-flex items-center rounded-lg px-2.5 py-1 text-xs font-medium"
                  style={{
                    backgroundColor: `color-mix(in srgb, ${effectiveColor} ${Math.round(opacity * 100)}%, transparent)`,
                    color: effectiveColor,
                    border: `1px solid color-mix(in srgb, ${effectiveColor} 30%, transparent)`,
                  }}
                >
                  <EditableField
                    value={skill.name}
                    onChange={(v) => updateSkill(skill.id, { name: v })}
                    placeholder="Skill"
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  {(skill.level || editable) && (
                    <>
                      <span className="mx-1 opacity-40">|</span>
                      <EditableSelect
                        value={skill.level}
                        onChange={(v) => updateSkill(skill.id, { level: v })}
                        options={SKILL_LEVEL_OPTIONS}
                        placeholder="Lvl"
                        className="text-[10px]"
                        editable={editable}
                      />
                    </>
                  )}
                </span>
              </EntryWrapper>
            );
          })}
        </div>
      );

    default:
      return null;
  }
}
