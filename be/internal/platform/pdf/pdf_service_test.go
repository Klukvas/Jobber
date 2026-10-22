package pdf

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLegacyResumePDFCannotBypassSectionContract(t *testing.T) {
	service, err := NewPDFService(zap.NewNop(), "")
	require.NoError(t, err)
	defer service.Close()
	for templateID, name := range templateIDMap {
		t.Run(name, func(t *testing.T) {
			data := &model.FullResumeDTO{
				ResumeBuilderDTO: &model.ResumeBuilderDTO{TemplateID: templateID, FontFamily: "Georgia", PrimaryColor: "#2563eb"},
				Experiences:      []*model.ExperienceDTO{{Company: "PRIVATE HIDDEN COMPANY"}},
				SectionOrder:     []*model.SectionOrderDTO{{SectionKey: "skills", SortOrder: 0, IsVisible: true}, {SectionKey: "experience", SortOrder: 1, IsVisible: false}},
			}
			result, err := service.GenerateResumePDF(context.Background(), data)
			require.ErrorContains(t, err, "React renderer is required")
			require.Empty(t, result, "unsupported legacy rendering must not return a successful document")
			require.Equal(t, "PRIVATE HIDDEN COMPANY", data.Experiences[0].Company)
			require.False(t, data.SectionOrder[1].IsVisible)
		})
	}
}
