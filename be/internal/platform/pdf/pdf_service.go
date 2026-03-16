package pdf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"go.uber.org/zap"
)

func float64Ptr(v float64) *float64 { return &v }

// pageStabilizeTimeoutMs is the time to wait for the page to stabilize before PDF generation.
const pageStabilizeTimeoutMs = 300

// colorRegex validates hex color values.
var pdfColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// allowedFonts is the set of valid font families for PDF rendering.
var allowedFonts = map[string]bool{
	"Georgia": true, "Arial": true, "Times New Roman": true,
	"Roboto": true, "Open Sans": true, "Lato": true, "Montserrat": true,
	"Poppins": true, "Inter": true, "Merriweather": true, "PT Serif": true,
	"Source Sans Pro": true, "Nunito": true, "Raleway": true, "Playfair Display": true,
}

// templateIDMap maps template UUIDs to template names.
// This is the single source of truth for template ID resolution.
var templateIDMap = map[string]string{
	"00000000-0000-0000-0000-000000000001": "professional",
	"00000000-0000-0000-0000-000000000002": "modern",
	"00000000-0000-0000-0000-000000000003": "minimal",
	"00000000-0000-0000-0000-000000000004": "executive",
	"00000000-0000-0000-0000-000000000005": "creative",
	"00000000-0000-0000-0000-000000000006": "compact",
	"00000000-0000-0000-0000-000000000007": "elegant",
	"00000000-0000-0000-0000-000000000008": "iconic",
	"00000000-0000-0000-0000-000000000009": "bold",
	"00000000-0000-0000-0000-00000000000a": "accent",
	"00000000-0000-0000-0000-00000000000b": "timeline",
	"00000000-0000-0000-0000-00000000000c": "vivid",
}

// CoverLetterPDFData holds the data needed to render a cover letter PDF.
type CoverLetterPDFData struct {
	Template       string
	FontFamily     string
	FontSize       int
	PrimaryColor   string
	RecipientName  string
	RecipientTitle string
	CompanyName    string
	CompanyAddress string
	Date           string
	Greeting       string
	Paragraphs     []string
	Closing        string
}

// frontendReadyTimeout is the max time to wait for the React print page to signal readiness.
const frontendReadyTimeout = 30 * time.Second

// frontendStabilizeDelay is additional wait time after __PDF_READY__ is set for paint to flush.
const frontendStabilizeDelay = 200 * time.Millisecond

// PDFService generates PDFs from resume data using headless Chrome via Rod.
type PDFService struct {
	logger      *zap.Logger
	frontendURL string
	browser     *rod.Browser
	mu          sync.RWMutex
	templates   map[string]*template.Template
	clTemplates map[string]*template.Template
}

// NewPDFService creates a new PDF service.
// It lazy-initializes the browser on first request.
// frontendURL enables React-based PDF rendering when non-empty.
func NewPDFService(logger *zap.Logger, frontendURL string) (*PDFService, error) {
	s := &PDFService{
		logger:      logger,
		frontendURL: frontendURL,
		templates:   make(map[string]*template.Template),
		clTemplates: make(map[string]*template.Template),
	}

	if err := s.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load PDF templates: %w", err)
	}

	return s, nil
}

func (s *PDFService) loadTemplates() error {
	funcMap := template.FuncMap{
		"formatDate":    formatDate,
		"dateRange":     dateRange,
		"safeHTML":      safeHTML,
		"joinSkills":    joinSkills,
		"joinLanguages": joinLanguages,
		"lightenColor":  lightenColor,
		"contrastColor": contrastColor,
		"safeURL":       safeURL,
	}

	for _, name := range templateIDMap {
		tmplContent, ok := embeddedTemplates[name]
		if !ok {
			return fmt.Errorf("template %q not found in embeddedTemplates", name)
		}
		tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplContent)
		if err != nil {
			return fmt.Errorf("failed to parse template %q: %w", name, err)
		}
		s.templates[name] = tmpl
	}

	// Load cover letter templates
	for name, tmplContent := range clEmbeddedTemplates {
		tmpl, err := template.New("cl_" + name).Parse(tmplContent)
		if err != nil {
			return fmt.Errorf("failed to parse cover letter template %q: %w", name, err)
		}
		s.clTemplates[name] = tmpl
	}

	return nil
}

func (s *PDFService) ensureBrowser() error {
	// Fast path: read lock to check if already initialized.
	s.mu.RLock()
	if s.browser != nil {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	// Slow path: write lock for initialization.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock.
	if s.browser != nil {
		return nil
	}

	path, found := launcher.LookPath()
	if !found {
		return fmt.Errorf("chromium not found in PATH")
	}

	u, err := launcher.New().Bin(path).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Headless(true).
		Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	s.browser = browser
	s.logger.Info("headless Chrome browser initialized for PDF generation")
	return nil
}

// acquireBrowser ensures the browser is initialized and returns with an RLock held.
// The caller MUST call s.mu.RUnlock() when done with the browser.
func (s *PDFService) acquireBrowser() error {
	if err := s.ensureBrowser(); err != nil {
		return err
	}
	s.mu.RLock()
	return nil
}

// sanitizeTemplateData returns a shallow copy of data with defense-in-depth sanitization
// applied to user-controlled CSS fields.
// SECURITY: html/template does NOT escape values inside <style> blocks, so these
// allowlist/regex checks are the sole barrier against CSS injection. Do not remove.
func sanitizeTemplateData(data *model.FullResumeDTO) *model.FullResumeDTO {
	// Shallow copy the embedded ResumeBuilderDTO via value copy.
	dto := *data.ResumeBuilderDTO
	out := *data
	out.ResumeBuilderDTO = &dto

	if !pdfColorRegex.MatchString(out.PrimaryColor) {
		out.PrimaryColor = "#2563eb"
	}
	if out.TextColor != "" && !pdfColorRegex.MatchString(out.TextColor) {
		out.TextColor = ""
	}
	if !allowedFonts[out.FontFamily] {
		out.FontFamily = "Georgia"
	}
	return &out
}

// GenerateResumePDF generates a PDF from resume data.
func (s *PDFService) GenerateResumePDF(ctx context.Context, data *model.FullResumeDTO) ([]byte, error) {
	if err := s.acquireBrowser(); err != nil {
		return nil, fmt.Errorf("browser init failed: %w", err)
	}
	defer s.mu.RUnlock()

	// Resolve template name from ID
	templateName := "professional"
	if name, ok := templateIDMap[data.TemplateID]; ok {
		templateName = name
	}

	tmpl, ok := s.templates[templateName]
	if !ok {
		return nil, fmt.Errorf("template %q not found", templateName)
	}

	// Defense-in-depth: sanitize user-controlled CSS values
	safe := sanitizeTemplateData(data)

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, safe); err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Propagate request context to Rod page operations
	page, err := s.browser.Context(ctx).Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	if err := page.SetDocumentContent(htmlBuf.String()); err != nil {
		return nil, fmt.Errorf("failed to set page content: %w", err)
	}

	if err := page.WaitStable(pageStabilizeTimeoutMs); err != nil {
		s.logger.Warn("page did not stabilize", zap.Error(err))
	}

	// Parse margins
	marginTop := 0.4
	marginBottom := 0.4
	marginLeft := 0.4
	marginRight := 0.4
	if safe.MarginTop > 0 {
		marginTop = float64(safe.MarginTop) / 96.0 // px to inches
	}
	if safe.MarginBottom > 0 {
		marginBottom = float64(safe.MarginBottom) / 96.0
	}
	if safe.MarginLeft > 0 {
		marginLeft = float64(safe.MarginLeft) / 96.0
	}
	if safe.MarginRight > 0 {
		marginRight = float64(safe.MarginRight) / 96.0
	}

	pdfData, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground: true,
		PaperWidth:      float64Ptr(8.27),  // A4
		PaperHeight:     float64Ptr(11.69), // A4
		MarginTop:       &marginTop,
		MarginBottom:    &marginBottom,
		MarginLeft:      &marginLeft,
		MarginRight:     &marginRight,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	result, err := io.ReadAll(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF bytes: %w", err)
	}

	return result, nil
}

// sanitizeCoverLetterData applies defense-in-depth sanitization on user-controlled fields.
func sanitizeCoverLetterData(data *CoverLetterPDFData) {
	if !pdfColorRegex.MatchString(data.PrimaryColor) {
		data.PrimaryColor = "#2563eb"
	}
	if !allowedFonts[data.FontFamily] {
		data.FontFamily = "Georgia"
	}
	if data.FontSize < 8 || data.FontSize > 18 {
		data.FontSize = 12
	}
}

// GenerateCoverLetterPDF generates a PDF from cover letter data.
func (s *PDFService) GenerateCoverLetterPDF(ctx context.Context, data *CoverLetterPDFData) ([]byte, error) {
	if err := s.acquireBrowser(); err != nil {
		return nil, fmt.Errorf("browser init failed: %w", err)
	}
	defer s.mu.RUnlock()

	templateName := data.Template
	if _, ok := s.clTemplates[templateName]; !ok {
		templateName = "professional"
	}

	tmpl := s.clTemplates[templateName]

	sanitizeCoverLetterData(data)

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, fmt.Errorf("failed to render cover letter template: %w", err)
	}

	page, err := s.browser.Context(ctx).Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	if err := page.SetDocumentContent(htmlBuf.String()); err != nil {
		return nil, fmt.Errorf("failed to set page content: %w", err)
	}

	if err := page.WaitStable(pageStabilizeTimeoutMs); err != nil {
		s.logger.Warn("page did not stabilize", zap.Error(err))
	}

	// A4 with 40px padding already in template — use minimal margins
	margin := 0.0

	pdfData, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground: true,
		PaperWidth:      float64Ptr(8.27),  // A4
		PaperHeight:     float64Ptr(11.69), // A4
		MarginTop:       &margin,
		MarginBottom:    &margin,
		MarginLeft:      &margin,
		MarginRight:     &margin,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	result, err := io.ReadAll(pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF bytes: %w", err)
	}

	return result, nil
}

// HasFrontendPDF returns true when the service is configured to render PDFs via the React frontend.
func (s *PDFService) HasFrontendPDF() bool {
	return s.frontendURL != ""
}

// GenerateResumePDFFromFrontend navigates headless Chrome to the frontend print route,
// injects resume data, waits for React to render, then captures a PDF.
// This guarantees a 1:1 match with what users see in the editor.
func (s *PDFService) GenerateResumePDFFromFrontend(ctx context.Context, data *model.FullResumeDTO) ([]byte, error) {
	if err := s.acquireBrowser(); err != nil {
		return nil, fmt.Errorf("browser init failed: %w", err)
	}
	defer s.mu.RUnlock()

	// Defense-in-depth: sanitize user-controlled CSS values before sending to frontend.
	safe := sanitizeTemplateData(data)

	jsonBytes, err := json.Marshal(safe)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resume data: %w", err)
	}

	page, err := s.browser.Context(ctx).Page(proto.TargetCreateTarget{
		URL: s.frontendURL + "/print/resume",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Wait for the SPA to boot (DOM content loaded).
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page load failed: %w", err)
	}

	// Inject resume data into window.__RESUME_DATA__ for the React print page to pick up.
	// SECURITY: Pass JSON as a string argument to prevent script injection via user content.
	if _, err := page.Eval(`(jsonStr) => { window.__RESUME_DATA__ = JSON.parse(jsonStr); }`, string(jsonBytes)); err != nil {
		return nil, fmt.Errorf("failed to inject resume data: %w", err)
	}

	// Poll for window.__PDF_READY__ === true with timeout.
	deadline := time.Now().Add(frontendReadyTimeout)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("frontend did not signal ready within %s", frontendReadyTimeout)
		}

		result, err := page.Eval(`() => window.__PDF_READY__ === true`)
		if err == nil && result.Value.Bool() {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Short stabilization delay for final paint flush.
	select {
	case <-time.After(frontendStabilizeDelay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Generate PDF with zero margins (the React page handles its own padding via CSS).
	margin := 0.0
	pdfReader, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground:    true,
		PreferCSSPageSize:  true,
		PaperWidth:         float64Ptr(8.27),  // A4
		PaperHeight:        float64Ptr(11.69), // A4
		MarginTop:          &margin,
		MarginBottom:       &margin,
		MarginLeft:         &margin,
		MarginRight:        &margin,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	result, err := io.ReadAll(pdfReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF bytes: %w", err)
	}

	s.logger.Info("generated PDF via frontend rendering",
		zap.String("template_id", safe.TemplateID),
		zap.Int("pdf_size", len(result)),
	)

	return result, nil
}

// Close shuts down the headless Chrome browser.
func (s *PDFService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.browser != nil {
		if err := s.browser.Close(); err != nil {
			s.logger.Error("failed to close browser", zap.Error(err))
		}
		s.browser = nil
	}
}
