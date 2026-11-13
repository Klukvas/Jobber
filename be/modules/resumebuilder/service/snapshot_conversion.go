package service

import "github.com/andreypavlenko/jobber/modules/resumebuilder/model"

type snapshotResumeV1 struct {
	*snapshotMetadataV1
	Contact        *snapshotContactV1         `json:"contact"`
	Summary        *snapshotSummaryV1         `json:"summary"`
	Experiences    []*snapshotExperienceV1    `json:"experiences"`
	Educations     []*snapshotEducationV1     `json:"educations"`
	Skills         []*snapshotSkillV1         `json:"skills"`
	Languages      []*snapshotLanguageV1      `json:"languages"`
	Certifications []*snapshotCertificationV1 `json:"certifications"`
	Projects       []*snapshotProjectV1       `json:"projects"`
	Volunteering   []*snapshotVolunteeringV1  `json:"volunteering"`
	CustomSections []*snapshotCustomSectionV1 `json:"custom_sections"`
	SectionOrder   []*snapshotSectionOrderV1  `json:"section_order"`
}

// Explicit casts fail to compile when API section/metadata fields change, forcing a storage-version decision.
func snapshotToStorage(full *model.FullResumeDTO) *snapshotResumeV1 {
	return &snapshotResumeV1{
		snapshotMetadataV1: convertSnapshotPointer(full.ResumeBuilderDTO, func(v model.ResumeBuilderDTO) snapshotMetadataV1 { return snapshotMetadataV1(v) }),
		Contact:            convertSnapshotPointer(full.Contact, func(v model.ContactDTO) snapshotContactV1 { return snapshotContactV1(v) }),
		Summary:            convertSnapshotPointer(full.Summary, func(v model.SummaryDTO) snapshotSummaryV1 { return snapshotSummaryV1(v) }),
		Experiences:        convertSnapshotSlice(full.Experiences, func(v model.ExperienceDTO) snapshotExperienceV1 { return snapshotExperienceV1(v) }),
		Educations:         convertSnapshotSlice(full.Educations, func(v model.EducationDTO) snapshotEducationV1 { return snapshotEducationV1(v) }),
		Skills:             convertSnapshotSlice(full.Skills, func(v model.SkillDTO) snapshotSkillV1 { return snapshotSkillV1(v) }),
		Languages:          convertSnapshotSlice(full.Languages, func(v model.LanguageDTO) snapshotLanguageV1 { return snapshotLanguageV1(v) }),
		Certifications:     convertSnapshotSlice(full.Certifications, func(v model.CertificationDTO) snapshotCertificationV1 { return snapshotCertificationV1(v) }),
		Projects:           convertSnapshotSlice(full.Projects, func(v model.ProjectDTO) snapshotProjectV1 { return snapshotProjectV1(v) }),
		Volunteering:       convertSnapshotSlice(full.Volunteering, func(v model.VolunteeringDTO) snapshotVolunteeringV1 { return snapshotVolunteeringV1(v) }),
		CustomSections:     convertSnapshotSlice(full.CustomSections, func(v model.CustomSectionDTO) snapshotCustomSectionV1 { return snapshotCustomSectionV1(v) }),
		SectionOrder:       convertSnapshotSlice(full.SectionOrder, func(v model.SectionOrderDTO) snapshotSectionOrderV1 { return snapshotSectionOrderV1(v) }),
	}
}

func snapshotFromStorage(full *snapshotResumeV1) *model.FullResumeDTO {
	return &model.FullResumeDTO{
		ResumeBuilderDTO: convertSnapshotPointer(full.snapshotMetadataV1, func(v snapshotMetadataV1) model.ResumeBuilderDTO { return model.ResumeBuilderDTO(v) }),
		Contact:          convertSnapshotPointer(full.Contact, func(v snapshotContactV1) model.ContactDTO { return model.ContactDTO(v) }),
		Summary:          convertSnapshotPointer(full.Summary, func(v snapshotSummaryV1) model.SummaryDTO { return model.SummaryDTO(v) }),
		Experiences:      convertSnapshotSlice(full.Experiences, func(v snapshotExperienceV1) model.ExperienceDTO { return model.ExperienceDTO(v) }),
		Educations:       convertSnapshotSlice(full.Educations, func(v snapshotEducationV1) model.EducationDTO { return model.EducationDTO(v) }),
		Skills:           convertSnapshotSlice(full.Skills, func(v snapshotSkillV1) model.SkillDTO { return model.SkillDTO(v) }),
		Languages:        convertSnapshotSlice(full.Languages, func(v snapshotLanguageV1) model.LanguageDTO { return model.LanguageDTO(v) }),
		Certifications:   convertSnapshotSlice(full.Certifications, func(v snapshotCertificationV1) model.CertificationDTO { return model.CertificationDTO(v) }),
		Projects:         convertSnapshotSlice(full.Projects, func(v snapshotProjectV1) model.ProjectDTO { return model.ProjectDTO(v) }),
		Volunteering:     convertSnapshotSlice(full.Volunteering, func(v snapshotVolunteeringV1) model.VolunteeringDTO { return model.VolunteeringDTO(v) }),
		CustomSections:   convertSnapshotSlice(full.CustomSections, func(v snapshotCustomSectionV1) model.CustomSectionDTO { return model.CustomSectionDTO(v) }),
		SectionOrder:     convertSnapshotSlice(full.SectionOrder, func(v snapshotSectionOrderV1) model.SectionOrderDTO { return model.SectionOrderDTO(v) }),
	}
}

func convertSnapshotPointer[From, To any](value *From, convert func(From) To) *To {
	if value == nil {
		return nil
	}
	copy := convert(*value)
	return &copy
}

func convertSnapshotSlice[From, To any](values []*From, convert func(From) To) []*To {
	if values == nil {
		return nil
	}
	result := make([]*To, len(values))
	for i, value := range values {
		result[i] = convertSnapshotPointer(value, convert)
	}
	return result
}
