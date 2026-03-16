package pdf

// embeddedTemplates holds the HTML templates for PDF generation.
var embeddedTemplates = map[string]string{
	"professional": professionalTemplate,
	"modern":       modernTemplate,
	"minimal":      minimalTemplate,
	"executive":    executiveTemplate,
	"creative":     creativeTemplate,
	"compact":      compactTemplate,
	"elegant":      elegantTemplate,
	"iconic":       iconicTemplate,
	"bold":         boldTemplate,
	"accent":       accentTemplate,
	"timeline":     timelineTemplate,
	"vivid":        vividTemplate,
}

// professionalTemplate is a single-column layout with centered name header,
// colored section headings, and bottom borders. Traditional corporate look.
const professionalTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }

  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }

  .container {
    padding: var(--spacing);
  }

  /* Header */
  .header {
    text-align: center;
    padding-bottom: calc(var(--spacing) * 0.8);
    margin-bottom: var(--spacing);
    border-bottom: 2px solid var(--primary);
  }

  .header h1 {
    font-size: 22pt;
    font-weight: 700;
    color: var(--primary);
    margin-bottom: 6px;
    letter-spacing: 0.5px;
  }

  .contact-row {
    display: flex;
    justify-content: center;
    flex-wrap: wrap;
    gap: 6px 16px;
    font-size: 9pt;
    color: #555555;
  }

  .contact-row a {
    color: #555555;
    text-decoration: none;
  }

  .contact-separator {
    color: #cccccc;
  }

  /* Sections */
  .section {
    margin-bottom: var(--spacing);
  }

  .section-title {
    font-size: 12pt;
    font-weight: 700;
    color: var(--primary);
    padding-bottom: 4px;
    border-bottom: 1px solid var(--primary);
    margin-bottom: calc(var(--spacing) * 0.5);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }

  .summary-text {
    font-size: 10pt;
    line-height: 1.6;
    color: #444444;
  }

  /* Entry items */
  .entry {
    margin-bottom: calc(var(--spacing) * 0.6);
    page-break-inside: avoid;
  }

  .entry-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 2px;
  }

  .entry-title {
    font-weight: 700;
    font-size: 10.5pt;
    color: #1a1a1a;
  }

  .entry-date {
    font-size: 9pt;
    color: #777777;
    white-space: nowrap;
    margin-left: 12px;
  }

  .entry-subtitle {
    font-size: 9.5pt;
    color: #555555;
    margin-bottom: 3px;
  }

  .entry-description {
    font-size: 9.5pt;
    line-height: 1.5;
    color: #444444;
  }

  /* Skills & Languages inline */
  .inline-list {
    font-size: 10pt;
    line-height: 1.7;
    color: #444444;
  }

  /* Certifications */
  .cert-link {
    color: var(--primary);
    text-decoration: none;
    font-size: 9pt;
  }

  .gpa {
    font-size: 9pt;
    color: #666666;
  }
</style>
</head>
<body>
<div class="container">

  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-row">
      {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}{{end}}
      {{if .Contact.LinkedIn}}{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}{{end}}
      {{if .Contact.GitHub}}{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}{{end}}
    </div>
  </div>
  {{end}}

  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">Summary</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}

  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="inline-list">{{joinSkills .Skills}}</div>
  </div>
  {{end}}

  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    <div class="inline-list">{{joinLanguages .Languages}}</div>
  </div>
  {{end}}

  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">
        {{if .Issuer}}{{.Issuer}}{{end}}
        {{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}
      </div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}

</div>
</body>
</html>`

// modernTemplate is a two-column layout with a colored left sidebar
// containing contact, skills, languages, and certifications. The right
// content area holds summary, experience, education, projects, and volunteering.
const modernTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --primary-light: {{lightenColor .PrimaryColor}};
    --primary-contrast: {{contrastColor .PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }

  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }

  .page {
    display: flex;
    min-height: 297mm;
  }

  /* Sidebar */
  .sidebar {
    width: 30%;
    background: var(--primary);
    color: var(--primary-contrast);
    padding: calc(var(--spacing) * 1.5) var(--spacing);
  }

  .sidebar h1 {
    font-size: 18pt;
    font-weight: 700;
    margin-bottom: 4px;
    line-height: 1.2;
    word-wrap: break-word;
  }

  .sidebar-section {
    margin-top: calc(var(--spacing) * 1.2);
  }

  .sidebar-title {
    font-size: 9pt;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1.2px;
    margin-bottom: 8px;
    opacity: 0.85;
    border-bottom: 1px solid rgba(255, 255, 255, 0.3);
    padding-bottom: 4px;
  }

  .sidebar-item {
    font-size: 9pt;
    margin-bottom: 5px;
    line-height: 1.4;
    word-wrap: break-word;
  }

  .sidebar-item a {
    color: var(--primary-contrast);
    text-decoration: none;
    opacity: 0.9;
  }

  .sidebar-label {
    font-size: 8pt;
    opacity: 0.7;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .skill-tag {
    display: inline-block;
    background: rgba(255, 255, 255, 0.15);
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 8.5pt;
    margin: 0 4px 4px 0;
  }

  .lang-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
    font-size: 9pt;
  }

  .lang-level {
    font-size: 8pt;
    opacity: 0.75;
  }

  .cert-item {
    margin-bottom: 8px;
  }

  .cert-name {
    font-size: 9pt;
    font-weight: 600;
  }

  .cert-issuer {
    font-size: 8pt;
    opacity: 0.75;
  }

  /* Content */
  .content {
    width: 70%;
    padding: calc(var(--spacing) * 1.5) calc(var(--spacing) * 1.2);
  }

  .section {
    margin-bottom: var(--spacing);
  }

  .section-title {
    font-size: 12pt;
    font-weight: 700;
    color: var(--primary);
    margin-bottom: calc(var(--spacing) * 0.4);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    border-bottom: 2px solid var(--primary-light);
    padding-bottom: 4px;
  }

  .summary-text {
    font-size: 10pt;
    line-height: 1.6;
    color: #444444;
  }

  .entry {
    margin-bottom: calc(var(--spacing) * 0.6);
    page-break-inside: avoid;
  }

  .entry-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 2px;
  }

  .entry-title {
    font-weight: 700;
    font-size: 10.5pt;
    color: #1a1a1a;
  }

  .entry-date {
    font-size: 9pt;
    color: #888888;
    white-space: nowrap;
    margin-left: 12px;
  }

  .entry-subtitle {
    font-size: 9.5pt;
    color: #555555;
    margin-bottom: 3px;
  }

  .entry-description {
    font-size: 9.5pt;
    line-height: 1.5;
    color: #444444;
  }

  .gpa {
    font-size: 9pt;
    color: #666666;
  }
</style>
</head>
<body>
<div class="page">

  <!-- Sidebar -->
  <div class="sidebar">

    {{if .Contact}}
    <h1>{{.Contact.FullName}}</h1>

    <div class="sidebar-section">
      <div class="sidebar-title">Contact</div>
      {{if .Contact.Email}}<div class="sidebar-item">{{.Contact.Email}}</div>{{end}}
      {{if .Contact.Phone}}<div class="sidebar-item">{{.Contact.Phone}}</div>{{end}}
      {{if .Contact.Location}}<div class="sidebar-item">{{.Contact.Location}}</div>{{end}}
      {{if .Contact.Website}}<div class="sidebar-item"><a href="{{.Contact.Website}}">{{.Contact.Website}}</a></div>{{end}}
      {{if .Contact.LinkedIn}}<div class="sidebar-item"><a href="{{.Contact.LinkedIn}}">LinkedIn</a></div>{{end}}
      {{if .Contact.GitHub}}<div class="sidebar-item"><a href="{{.Contact.GitHub}}">GitHub</a></div>{{end}}
    </div>
    {{end}}

    {{if gt (len .Skills) 0}}
    <div class="sidebar-section">
      <div class="sidebar-title">Skills</div>
      <div>
        {{range .Skills}}
        <span class="skill-tag">{{.Name}}</span>
        {{end}}
      </div>
    </div>
    {{end}}

    {{if gt (len .Languages) 0}}
    <div class="sidebar-section">
      <div class="sidebar-title">Languages</div>
      {{range .Languages}}
      <div class="lang-item">
        <span>{{.Name}}</span>
        {{if .Proficiency}}<span class="lang-level">{{.Proficiency}}</span>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if gt (len .Certifications) 0}}
    <div class="sidebar-section">
      <div class="sidebar-title">Certifications</div>
      {{range .Certifications}}
      <div class="cert-item">
        <div class="cert-name">{{.Name}}</div>
        {{if .Issuer}}<div class="cert-issuer">{{.Issuer}}</div>{{end}}
        {{if .IssueDate}}<div class="cert-issuer">{{formatDate .IssueDate}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

  </div>

  <!-- Main Content -->
  <div class="content">

    {{if and .Summary .Summary.Content}}
    <div class="section">
      <div class="section-title">Summary</div>
      <div class="summary-text">{{safeHTML .Summary.Content}}</div>
    </div>
    {{end}}

    {{if gt (len .Experiences) 0}}
    <div class="section">
      <div class="section-title">Experience</div>
      {{range .Experiences}}
      <div class="entry">
        <div class="entry-header">
          <span class="entry-title">{{.Position}}</span>
          <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
        </div>
        <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
        {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if gt (len .Educations) 0}}
    <div class="section">
      <div class="section-title">Education</div>
      {{range .Educations}}
      <div class="entry">
        <div class="entry-header">
          <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
          <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
        </div>
        <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
        {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if gt (len .Projects) 0}}
    <div class="section">
      <div class="section-title">Projects</div>
      {{range .Projects}}
      <div class="entry">
        <div class="entry-header">
          <span class="entry-title">{{.Name}}</span>
          <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
        </div>
        {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
        {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if gt (len .Volunteering) 0}}
    <div class="section">
      <div class="section-title">Volunteering</div>
      {{range .Volunteering}}
      <div class="entry">
        <div class="entry-header">
          <span class="entry-title">{{.Role}}</span>
          <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
        </div>
        <div class="entry-subtitle">{{.Organization}}</div>
        {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}

    {{if gt (len .CustomSections) 0}}
    {{range .CustomSections}}
    <div class="section">
      <div class="section-title">{{.Title}}</div>
      <div class="entry-description">{{safeHTML .Content}}</div>
    </div>
    {{end}}
    {{end}}

  </div>

</div>
</body>
</html>`

// minimalTemplate is a single-column, whitespace-heavy layout with minimal
// decoration. Thin primary color line under header, uppercase small section
// headings, and generous spacing for a clean modern aesthetic.
const minimalTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }

  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.6;
    color: #333333;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }

  .container {
    padding: var(--spacing);
  }

  /* Header */
  .header {
    margin-bottom: calc(var(--spacing) * 0.6);
  }

  .header h1 {
    font-size: 24pt;
    font-weight: 300;
    color: #1a1a1a;
    letter-spacing: 1px;
    margin-bottom: 4px;
  }

  .contact-line {
    font-size: 8.5pt;
    color: #888888;
    letter-spacing: 0.3px;
  }

  .contact-line a {
    color: #888888;
    text-decoration: none;
  }

  .contact-line span + span::before {
    content: "  |  ";
    color: #cccccc;
  }

  .header-line {
    width: 100%;
    height: 1px;
    background: var(--primary);
    margin-top: calc(var(--spacing) * 0.5);
  }

  /* Sections */
  .section {
    margin-top: calc(var(--spacing) * 0.9);
  }

  .section-title {
    font-size: 8pt;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 2px;
    color: var(--primary);
    margin-bottom: calc(var(--spacing) * 0.4);
  }

  .summary-text {
    font-size: 9.5pt;
    line-height: 1.7;
    color: #555555;
    font-weight: 300;
  }

  /* Entry items */
  .entry {
    margin-bottom: calc(var(--spacing) * 0.6);
    page-break-inside: avoid;
  }

  .entry-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .entry-title {
    font-weight: 600;
    font-size: 10pt;
    color: #1a1a1a;
  }

  .entry-date {
    font-size: 8.5pt;
    color: #999999;
    white-space: nowrap;
    margin-left: 16px;
    font-weight: 300;
  }

  .entry-subtitle {
    font-size: 9pt;
    color: #777777;
    font-weight: 300;
    margin-bottom: 2px;
  }

  .entry-description {
    font-size: 9pt;
    line-height: 1.6;
    color: #555555;
    font-weight: 300;
    margin-top: 2px;
  }

  /* Inline lists */
  .inline-list {
    font-size: 9.5pt;
    line-height: 1.8;
    color: #555555;
    font-weight: 300;
  }

  .gpa {
    font-size: 8.5pt;
    color: #999999;
    font-weight: 300;
  }

  .cert-link {
    color: var(--primary);
    text-decoration: none;
    font-size: 8.5pt;
    font-weight: 300;
  }

  .project-url {
    color: var(--primary);
    text-decoration: none;
    font-size: 8.5pt;
    font-weight: 300;
  }
</style>
</head>
<body>
<div class="container">

  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-line">
      {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}<span><a href="{{.Contact.Website}}">{{.Contact.Website}}</a></span>{{end}}
      {{if .Contact.LinkedIn}}<span><a href="{{.Contact.LinkedIn}}">LinkedIn</a></span>{{end}}
      {{if .Contact.GitHub}}<span><a href="{{.Contact.GitHub}}">GitHub</a></span>{{end}}
    </div>
    <div class="header-line"></div>
  </div>
  {{end}}

  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">Summary</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}

  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-row">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-row">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="inline-list">{{joinSkills .Skills}}</div>
  </div>
  {{end}}

  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    <div class="inline-list">{{joinLanguages .Languages}}</div>
  </div>
  {{end}}

  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-row">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">
        {{if .Issuer}}{{.Issuer}}{{end}}
        {{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}
      </div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-row">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a class="project-url" href="{{.}}">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-row">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}

</div>
</body>
</html>`

// executiveTemplate is a formal corporate layout with a dark header bar,
// small-caps section headings, and thin bottom borders. Conservative look
// for senior and management roles.
const executiveTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }
  .container { padding: var(--spacing); }
  .header {
    background: var(--primary);
    color: #ffffff;
    padding: calc(var(--spacing) * 1.2) calc(var(--spacing) * 1.5);
    margin-bottom: var(--spacing);
  }
  .header h1 { font-size: 22pt; font-weight: 700; letter-spacing: 1px; margin-bottom: 6px; }
  .contact-row { display: flex; flex-wrap: wrap; gap: 6px 18px; font-size: 9pt; opacity: 0.85; }
  .contact-row a { color: #ffffff; text-decoration: none; opacity: 0.85; }
  .section { margin-bottom: var(--spacing); }
  .section-title {
    font-size: 10pt; font-weight: 600; color: #444444;
    padding-bottom: 4px; border-bottom: 1px solid #cccccc;
    margin-bottom: calc(var(--spacing) * 0.5);
    text-transform: uppercase; letter-spacing: 2.5px; font-variant: small-caps;
  }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444444; }
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #777777; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444444; }
  .inline-list { font-size: 10pt; line-height: 1.7; color: #444444; }
  .gpa { font-size: 9pt; color: #666666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-row">
      {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}{{end}}
      {{if .Contact.LinkedIn}}{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}{{end}}
      {{if .Contact.GitHub}}{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}{{end}}
    </div>
  </div>
  {{end}}
<div class="container">
  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">Executive Summary</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}
  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="inline-list">{{joinSkills .Skills}}</div>
  </div>
  {{end}}
  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    <div class="inline-list">{{joinLanguages .Languages}}</div>
  </div>
  {{end}}
  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">
        {{if .Issuer}}{{.Issuer}}{{end}}
        {{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}
      </div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}
</div>
</body>
</html>`

// creativeTemplate is a bold, colorful layout with an initials circle,
// thick left-border section headings, tag-style skills, and a modern
// asymmetric feel for design and marketing roles.
const creativeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --primary-light: {{lightenColor .PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }
  .container { padding: var(--spacing); }
  .header { display: flex; align-items: center; gap: 16px; margin-bottom: var(--spacing); }
  .initials {
    width: 56px; height: 56px; border-radius: 50%;
    background: var(--primary); color: #ffffff;
    font-size: 18pt; font-weight: 700;
    display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  }
  .header h1 { font-size: 22pt; font-weight: 700; color: var(--primary); margin-bottom: 2px; }
  .contact-row { display: flex; flex-wrap: wrap; gap: 4px 14px; font-size: 9pt; color: #666666; }
  .contact-row a { color: #666666; text-decoration: none; }
  .section { margin-bottom: var(--spacing); }
  .section-title {
    font-size: 12pt; font-weight: 700; color: var(--primary);
    padding-left: 10px; border-left: 4px solid var(--primary);
    margin-bottom: calc(var(--spacing) * 0.5);
  }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444444; }
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #888888; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: var(--primary); margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444444; }
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag {
    display: inline-block; background: var(--primary); color: #ffffff;
    padding: 2px 10px; border-radius: 12px; font-size: 9pt;
  }
  .lang-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; font-size: 10pt; color: #444444; }
  .lang-level { font-size: 9pt; color: #888888; }
  .gpa { font-size: 9pt; color: #666666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
<div class="container">
  {{if .Contact}}
  <div class="header">
    <div class="initials">{{with .Contact.FullName}}{{slice . 0 1}}{{end}}</div>
    <div>
      <h1>{{.Contact.FullName}}</h1>
      <div class="contact-row">
        {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
        {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
        {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
        {{if .Contact.Website}}{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}{{end}}
        {{if .Contact.LinkedIn}}{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}{{end}}
        {{if .Contact.GitHub}}{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}{{end}}
      </div>
    </div>
  </div>
  {{end}}
  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">About Me</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}
  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="skill-tags">
      {{range .Skills}}<span class="skill-tag">{{.Name}}{{if .Level}} ({{.Level}}){{end}}</span>{{end}}
    </div>
  </div>
  {{end}}
  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    {{range .Languages}}
    <div class="lang-row">
      <span>{{.Name}}</span>
      {{if .Proficiency}}<span class="lang-level">{{.Proficiency}}</span>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">
        {{if .Issuer}}{{.Issuer}}{{end}}
        {{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}
      </div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}
</div>
</body>
</html>`

// compactTemplate is a dense, information-packed layout with small fonts,
// tight spacing, multi-column skills grid, and a single-line header.
// Designed for academic and technical resumes that need maximum content.
const compactTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --font: {{.FontFamily}}, sans-serif;
    --spacing: {{.Spacing}}px;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: var(--font);
    font-size: 8.5pt;
    line-height: 1.35;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }
  .container { padding: calc(var(--spacing) * 0.7); }
  .header {
    display: flex; justify-content: space-between; align-items: baseline;
    padding-bottom: 6px; border-bottom: 2px solid var(--primary);
    margin-bottom: calc(var(--spacing) * 0.6);
  }
  .header h1 { font-size: 16pt; font-weight: 700; color: var(--primary); }
  .contact-row {
    display: flex; flex-wrap: wrap; justify-content: flex-end;
    gap: 3px 10px; font-size: 7.5pt; color: #666666; max-width: 55%;
  }
  .contact-row a { color: #666666; text-decoration: none; }
  .section { margin-bottom: calc(var(--spacing) * 0.6); }
  .section-title {
    font-size: 8.5pt; font-weight: 700; color: var(--primary);
    text-transform: uppercase; letter-spacing: 1.5px;
    margin-bottom: 3px; border-bottom: 0.5px solid #dddddd; padding-bottom: 2px;
  }
  .summary-text { font-size: 8.5pt; line-height: 1.4; color: #444444; }
  .entry { margin-bottom: calc(var(--spacing) * 0.35); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 1px; }
  .entry-title { font-weight: 700; font-size: 9pt; color: #1a1a1a; }
  .entry-date { font-size: 7.5pt; color: #888888; white-space: nowrap; margin-left: 8px; }
  .entry-subtitle { font-size: 8pt; color: #555555; margin-bottom: 1px; }
  .entry-description { font-size: 8.5pt; line-height: 1.35; color: #444444; }
  .multi-col {
    display: grid; grid-template-columns: repeat(3, 1fr);
    gap: 1px 16px; font-size: 8.5pt; color: #444444;
  }
  .multi-col-item { line-height: 1.5; }
  .gpa { font-size: 7.5pt; color: #666666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 7.5pt; }
</style>
</head>
<body>
<div class="container">
  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-row">
      {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}{{end}}
      {{if .Contact.LinkedIn}}{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}{{end}}
      {{if .Contact.GitHub}}{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}{{end}}
    </div>
  </div>
  {{end}}
  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">Summary</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}
  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="multi-col">
      {{range .Skills}}<div class="multi-col-item">{{.Name}}{{if .Level}} ({{.Level}}){{end}}</div>{{end}}
    </div>
  </div>
  {{end}}
  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    <div class="multi-col">
      {{range .Languages}}<div class="multi-col-item">{{.Name}}{{if .Proficiency}} &mdash; {{.Proficiency}}{{end}}</div>{{end}}
    </div>
  </div>
  {{end}}
  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">
        {{if .Issuer}}{{.Issuer}}{{end}}
        {{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}
      </div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 7.5pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}
</div>
</body>
</html>`

// elegantTemplate is a polished single-column layout with diamond bullet
// icons before section headers and pill-style skills tags.
const elegantTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root { --primary: {{.PrimaryColor}}; --font: {{.FontFamily}}, sans-serif; --spacing: {{.Spacing}}px; }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: var(--font); font-size: 10pt; line-height: 1.5; color: #2d2d2d; width: 210mm; min-height: 297mm; background: #fff; }
  .container { padding: var(--spacing); }
  .header { margin-bottom: var(--spacing); }
  .header h1 { font-size: 22pt; font-weight: 700; color: var(--primary); margin-bottom: 4px; }
  .contact-line { font-size: 9pt; color: #777; }
  .contact-line span + span::before { content: " | "; color: #ccc; }
  .contact-line a { color: #777; text-decoration: none; }
  .header-line { width: 100%; height: 2px; background: var(--primary); margin-top: 8px; }
  .section { margin-bottom: var(--spacing); }
  .section-title { font-size: 11pt; font-weight: 700; color: var(--primary); margin-bottom: calc(var(--spacing) * 0.4); }
  .section-title::before { content: "\25C6  "; }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #888; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag { display: inline-block; background: var(--primary); color: #fff; padding: 2px 10px; border-radius: 12px; font-size: 9pt; }
  .inline-list { font-size: 10pt; line-height: 1.7; color: #444; }
  .gpa { font-size: 9pt; color: #666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
<div class="container">
  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-line">
      {{if .Contact.Email}}<span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}<span>{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}</span>{{end}}
      {{if .Contact.LinkedIn}}<span>{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}</span>{{end}}
      {{if .Contact.GitHub}}<span>{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}</span>{{end}}
    </div>
    <div class="header-line"></div>
  </div>
  {{end}}
  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-title">Summary</div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}
  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-title">Work Experience</div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-title">Education</div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-title">Skills</div>
    <div class="skill-tags">
      {{range .Skills}}<span class="skill-tag">{{.Name}}{{if .Level}} ({{.Level}}){{end}}</span>{{end}}
    </div>
  </div>
  {{end}}
  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-title">Languages</div>
    <div class="inline-list">{{joinLanguages .Languages}}</div>
  </div>
  {{end}}
  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-title">Certifications</div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">{{if .Issuer}}{{.Issuer}}{{end}}{{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}</div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-title">Projects</div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-title">Volunteering</div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-title">{{.Title}}</div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}
</div>
</body>
</html>`

// iconicTemplate is a visual layout with colored circle icons before each
// section header and contact info with small icon badges.
const iconicTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root { --primary: {{.PrimaryColor}}; --font: {{.FontFamily}}, sans-serif; --spacing: {{.Spacing}}px; }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: var(--font); font-size: 10pt; line-height: 1.5; color: #2d2d2d; width: 210mm; min-height: 297mm; background: #fff; }
  .container { padding: var(--spacing); }
  .header { margin-bottom: var(--spacing); }
  .header h1 { font-size: 22pt; font-weight: 700; color: var(--primary); margin-bottom: 8px; }
  .contact-row { display: flex; flex-wrap: wrap; gap: 6px 16px; font-size: 9pt; color: #666; }
  .contact-row a { color: #666; text-decoration: none; }
  .contact-icon { display: inline-flex; align-items: center; justify-content: center; width: 16px; height: 16px; border-radius: 50%; background: var(--primary); color: #fff; font-size: 8px; margin-right: 4px; vertical-align: middle; }
  .section { margin-bottom: var(--spacing); }
  .section-header { display: flex; align-items: center; margin-bottom: calc(var(--spacing) * 0.5); }
  .section-icon { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border-radius: 50%; background: var(--primary); color: #fff; font-size: 12px; margin-right: 8px; flex-shrink: 0; }
  .section-title { font-size: 12pt; font-weight: 700; color: var(--primary); }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #888; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: var(--primary); margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag { display: inline-block; background: var(--primary); color: #fff; padding: 2px 10px; border-radius: 12px; font-size: 9pt; }
  .lang-row { display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 10pt; color: #444; }
  .lang-level { font-size: 9pt; color: #888; }
  .gpa { font-size: 9pt; color: #666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
<div class="container">
  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="contact-row">
      {{if .Contact.Email}}<span><span class="contact-icon">&#9993;</span>{{.Contact.Email}}</span>{{end}}
      {{if .Contact.Phone}}<span><span class="contact-icon">&#9742;</span>{{.Contact.Phone}}</span>{{end}}
      {{if .Contact.Location}}<span><span class="contact-icon">&#9906;</span>{{.Contact.Location}}</span>{{end}}
      {{if .Contact.Website}}<span><span class="contact-icon">&#9737;</span>{{with safeURL .Contact.Website}}<a href="{{.}}">{{$.Contact.Website}}</a>{{end}}</span>{{end}}
      {{if .Contact.LinkedIn}}<span><span class="contact-icon">in</span>{{with safeURL .Contact.LinkedIn}}<a href="{{.}}">LinkedIn</a>{{end}}</span>{{end}}
      {{if .Contact.GitHub}}<span><span class="contact-icon">&#10070;</span>{{with safeURL .Contact.GitHub}}<a href="{{.}}">GitHub</a>{{end}}</span>{{end}}
    </div>
  </div>
  {{end}}
  {{if and .Summary .Summary.Content}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9998;</span><span class="section-title">About Me</span></div>
    <div class="summary-text">{{safeHTML .Summary.Content}}</div>
  </div>
  {{end}}
  {{if gt (len .Experiences) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9881;</span><span class="section-title">Work Experience</span></div>
    {{range .Experiences}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Position}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Company}}{{if .Location}} &mdash; {{.Location}}{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Educations) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9734;</span><span class="section-title">Education</span></div>
    {{range .Educations}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Degree}}{{if .FieldOfStudy}} in {{.FieldOfStudy}}{{end}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate .IsCurrent}}</span>
      </div>
      <div class="entry-subtitle">{{.Institution}}{{if .GPA}} <span class="gpa">&bull; GPA: {{.GPA}}</span>{{end}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Skills) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9889;</span><span class="section-title">Skills</span></div>
    <div class="skill-tags">
      {{range .Skills}}<span class="skill-tag">{{.Name}}{{if .Level}} ({{.Level}}){{end}}</span>{{end}}
    </div>
  </div>
  {{end}}
  {{if gt (len .Languages) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9742;</span><span class="section-title">Languages</span></div>
    {{range .Languages}}
    <div class="lang-row">
      <span>{{.Name}}</span>
      {{if .Proficiency}}<span class="lang-level">{{.Proficiency}}</span>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Certifications) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#10003;</span><span class="section-title">Certifications</span></div>
    {{range .Certifications}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{if .IssueDate}}{{formatDate .IssueDate}}{{end}}</span>
      </div>
      <div class="entry-subtitle">{{if .Issuer}}{{.Issuer}}{{end}}{{if .ExpiryDate}} &bull; Expires: {{formatDate .ExpiryDate}}{{end}}</div>
      {{if .URL}}{{with safeURL .URL}}<a class="cert-link" href="{{.}}">View credential</a>{{end}}{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Projects) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9733;</span><span class="section-title">Projects</span></div>
    {{range .Projects}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Name}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      {{if .URL}}{{$url := .URL}}{{with safeURL .URL}}<div class="entry-subtitle"><a href="{{.}}" style="color: var(--primary); text-decoration: none; font-size: 9pt;">{{$url}}</a></div>{{end}}{{end}}
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .Volunteering) 0}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#10084;</span><span class="section-title">Volunteering</span></div>
    {{range .Volunteering}}
    <div class="entry">
      <div class="entry-header">
        <span class="entry-title">{{.Role}}</span>
        <span class="entry-date">{{dateRange .StartDate .EndDate false}}</span>
      </div>
      <div class="entry-subtitle">{{.Organization}}</div>
      {{if .Description}}<div class="entry-description">{{safeHTML .Description}}</div>{{end}}
    </div>
    {{end}}
  </div>
  {{end}}
  {{if gt (len .CustomSections) 0}}
  {{range .CustomSections}}
  <div class="section">
    <div class="section-header"><span class="section-icon">&#9776;</span><span class="section-title">{{.Title}}</span></div>
    <div class="entry-description">{{safeHTML .Content}}</div>
  </div>
  {{end}}
  {{end}}
</div>
</body>
</html>`
