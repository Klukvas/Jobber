//go:build integration

package pdf

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/go-rod/rod/lib/devices"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// This uses the production headless export path, with a fresh browser and no
// stored consent. Run against an already running local frontend.
func TestFrontendResumePDFContent(t *testing.T) {
	frontendURL := os.Getenv("FRONTEND_PDF_TEST_URL")
	if frontendURL == "" {
		t.Skip("set FRONTEND_PDF_TEST_URL to run real PDF rendering")
	}
	pdftotext, err := exec.LookPath("pdftotext")
	require.NoError(t, err)
	for _, device := range []struct {
		name   string
		device devices.Device
	}{{"desktop", devices.Clear}, {"mobile", devices.IPhoneX}} {
		t.Run(device.name, func(t *testing.T) {
			service, err := NewPDFService(zap.NewNop(), frontendURL)
			require.NoError(t, err)
			require.NoError(t, service.ensureBrowser())
			service.browser = service.browser.DefaultDevice(device.device)
			defer service.Close()
			for templateID, name := range templateIDMap {
				t.Run(name, func(t *testing.T) {
					data := &model.FullResumeDTO{
						ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "synthetic-print", TemplateID: templateID, FontFamily: "Georgia", PrimaryColor: "#2563eb", FontSize: 12, Spacing: 115, MarginTop: 40, MarginBottom: 40, MarginLeft: 40, MarginRight: 40, LayoutMode: "single", SidebarWidth: 35},
						Contact:          &model.ContactDTO{FullName: "Synthetic PDF Candidate"},
						Summary:          &model.SummaryDTO{Content: "Summary Content Marker"},
						Experiences:      []*model.ExperienceDTO{{ID: "hidden", Company: "PRIVATE HIDDEN COMPANY"}},
						Educations:       []*model.EducationDTO{{ID: "partial", Institution: "Partial Education Marker"}},
						Skills:           []*model.SkillDTO{{ID: "skill", Name: "Skill Content Marker"}},
						Languages:        []*model.LanguageDTO{}, Certifications: []*model.CertificationDTO{}, Projects: []*model.ProjectDTO{}, Volunteering: []*model.VolunteeringDTO{}, CustomSections: []*model.CustomSectionDTO{},
						SectionOrder: []*model.SectionOrderDTO{
							{SectionKey: "skills", SortOrder: 0, IsVisible: true, Column: "main"},
							{SectionKey: "education", SortOrder: 1, IsVisible: true, Column: "main"},
							{SectionKey: "summary", SortOrder: 2, IsVisible: true, Column: "main"},
							{SectionKey: "experience", SortOrder: 3, IsVisible: false, Column: "main"},
						},
					}
					ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
					defer cancel()
					pdfBytes, err := service.GenerateResumePDFFromFrontend(ctx, data)
					require.NoError(t, err)
					require.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")))
					require.Greater(t, len(pdfBytes), 1000)
					command := exec.CommandContext(ctx, pdftotext, "-", "-")
					command.Stdin = bytes.NewReader(pdfBytes)
					textBytes, err := command.Output()
					require.NoError(t, err)
					text := string(textBytes)
					for _, unwanted := range []string{"PRIVATE HIDDEN COMPANY", "Add dates", "Position", "Degree", "Your Name", "We use essential cookies", "Privacy Policy", "Essential only", "Accept all"} {
						require.NotContains(t, text, unwanted)
					}
					for _, wanted := range []string{"Synthetic PDF Candidate", "Skill Content Marker", "Partial Education Marker", "Summary Content Marker"} {
						require.Contains(t, text, wanted)
					}
					require.Less(t, strings.Index(text, "Skill Content Marker"), strings.Index(text, "Partial Education Marker"))
					require.Less(t, strings.Index(text, "Partial Education Marker"), strings.Index(text, "Summary Content Marker"))
					if outputDir := os.Getenv("PDF_TEST_ARTIFACT_DIR"); outputDir != "" {
						require.NoError(t, os.MkdirAll(outputDir, 0755))
						filename := filepath.Join(outputDir, device.name+"-"+name)
						require.NoError(t, os.WriteFile(filename+".pdf", pdfBytes, 0644))
						require.NoError(t, os.WriteFile(filename+".txt", textBytes, 0644))
					}
					// A visible, incomplete experience must keep the user's company
					// without printing Position or Add dates. Keep the original
					// hidden-section fixture intact for the assertion above.
					partial := *data
					partial.Experiences = []*model.ExperienceDTO{{ID: "partial-experience", Company: "Partial Company Marker"}}
					partial.SectionOrder = append([]*model.SectionOrderDTO{}, data.SectionOrder[:3]...)
					partial.SectionOrder = append(partial.SectionOrder, &model.SectionOrderDTO{SectionKey: "experience", SortOrder: 3, IsVisible: true, Column: "main"})
					partialBytes, err := service.GenerateResumePDFFromFrontend(ctx, &partial)
					require.NoError(t, err)
					partialCommand := exec.CommandContext(ctx, pdftotext, "-", "-")
					partialCommand.Stdin = bytes.NewReader(partialBytes)
					partialText, err := partialCommand.Output()
					require.NoError(t, err)
					require.Contains(t, string(partialText), "Partial Company Marker")
					for _, unwanted := range []string{"Position", "Add dates", "PRIVATE HIDDEN COMPANY", "We use essential cookies", "Privacy Policy"} {
						require.NotContains(t, string(partialText), unwanted)
					}
					if outputDir := os.Getenv("PDF_TEST_ARTIFACT_DIR"); outputDir != "" {
						filename := filepath.Join(outputDir, device.name+"-"+name+"-partial")
						require.NoError(t, os.WriteFile(filename+".pdf", partialBytes, 0644))
						require.NoError(t, os.WriteFile(filename+".txt", partialText, 0644))
					}

					if device.name == "desktop" {
						blank := *data
						blank.Contact = &model.ContactDTO{}
						blank.Summary = &model.SummaryDTO{Content: " \n "}
						blank.Experiences = []*model.ExperienceDTO{{ID: "blank"}}
						blank.Educations = []*model.EducationDTO{{ID: "blank"}}
						blank.Skills = []*model.SkillDTO{{ID: "blank"}}
						blank.SectionOrder = []*model.SectionOrderDTO{}
						for index, key := range []string{"contact", "summary", "experience", "education", "skills", "languages", "certifications", "projects", "volunteering", "custom"} {
							blank.SectionOrder = append(blank.SectionOrder, &model.SectionOrderDTO{SectionKey: key, SortOrder: index, IsVisible: true, Column: "main"})
						}
						blankBytes, err := service.GenerateResumePDFFromFrontend(ctx, &blank)
						require.NoError(t, err)
						blankCommand := exec.CommandContext(ctx, pdftotext, "-", "-")
						blankCommand.Stdin = bytes.NewReader(blankBytes)
						blankText, err := blankCommand.Output()
						require.NoError(t, err)
						require.Empty(t, strings.TrimSpace(string(blankText)), "default or blank rows must not print section headings or placeholders")
						if outputDir := os.Getenv("PDF_TEST_ARTIFACT_DIR"); outputDir != "" {
							require.NoError(t, os.WriteFile(filepath.Join(outputDir, device.name+"-"+name+"-blank.pdf"), blankBytes, 0644))
						}
					}
					t.Logf("valid PDF bytes=%d partial=%d; hidden/placeholder/consent absent; section order correct", len(pdfBytes), len(partialBytes))
				})
			}
		})
	}
}
