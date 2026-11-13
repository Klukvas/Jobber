package service

import (
	"bytes"
	"encoding/json"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

const snapshotFormatVersion = 1

type snapshotEnvelope struct {
	FormatVersion int               `json:"format_version"`
	Resume        *snapshotResumeV1 `json:"resume"`
}

func encodeResumeSnapshot(full *model.FullResumeDTO) (json.RawMessage, error) {
	if full == nil || full.ResumeBuilderDTO == nil {
		return nil, model.ErrInvalidSnapshotContent
	}
	return json.Marshal(snapshotEnvelope{FormatVersion: snapshotFormatVersion, Resume: snapshotToStorage(full)})
}

func decodeResumeSnapshot(snapshot *model.ResumeSnapshot) (*model.FullResumeDTO, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(snapshot.Content, &fields); err != nil || fields == nil {
		return nil, model.ErrInvalidSnapshotContent
	}
	data := snapshot.Content
	stored := legacySnapshotDefaults()
	if version, versioned := fields["format_version"]; versioned {
		var number int
		if err := json.Unmarshal(version, &number); err != nil || number != snapshotFormatVersion {
			return nil, model.ErrUnsupportedSnapshotVersion
		}
		if len(fields) != 2 || !completeSnapshotV1(fields["resume"]) {
			return nil, model.ErrInvalidSnapshotContent
		}
		data = fields["resume"]
		stored = &snapshotResumeV1{snapshotMetadataV1: &snapshotMetadataV1{}}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(stored); err != nil || stored == nil || stored.snapshotMetadataV1 == nil {
		return nil, model.ErrInvalidSnapshotContent
	}
	if stored.ID != snapshot.ResumeBuilderID || !validSnapshotSections(stored) {
		return nil, model.ErrInvalidSnapshotContent
	}
	return snapshotFromStorage(stored), nil
}

// Unversioned snapshots are v0. Only absent fields get defaults; explicit zero
// margins and empty skill display remain intact. Future migrations belong here.
func legacySnapshotDefaults() *snapshotResumeV1 {
	return &snapshotResumeV1{snapshotMetadataV1: &snapshotMetadataV1{
		Title: "Untitled Resume", TemplateID: DefaultTemplateID, FontFamily: DefaultFontFamily,
		PrimaryColor: DefaultAccentColor, TextColor: DefaultTextColor, Spacing: DefaultSpacing,
		MarginTop: DefaultMargin, MarginBottom: DefaultMargin, MarginLeft: DefaultMargin, MarginRight: DefaultMargin,
		LayoutMode: DefaultLayoutMode, SidebarWidth: DefaultSidebarWidth, FontSize: DefaultFontSize,
	}}
}

func validSnapshotSections(full *snapshotResumeV1) bool {
	return noNilSnapshotEntries(full.Experiences) && noNilSnapshotEntries(full.Educations) &&
		noNilSnapshotEntries(full.Skills) && noNilSnapshotEntries(full.Languages) &&
		noNilSnapshotEntries(full.Certifications) && noNilSnapshotEntries(full.Projects) &&
		noNilSnapshotEntries(full.Volunteering) && noNilSnapshotEntries(full.CustomSections) &&
		noNilSnapshotEntries(full.SectionOrder)
}

func noNilSnapshotEntries[T any](entries []*T) bool {
	for _, entry := range entries {
		if entry == nil {
			return false
		}
	}
	return true
}
