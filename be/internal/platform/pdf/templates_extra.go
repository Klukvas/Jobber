package pdf

// boldTemplate is a high-contrast layout with a full-width colored header
// banner and pill-style skills.
const boldTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --primary-contrast: {{contrastColor .PrimaryColor}};
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

  /* Full-width colored header */
  .header {
    background: var(--primary);
    color: var(--primary-contrast);
    padding: calc(var(--spacing) * 1.5) var(--spacing);
    border-radius: 6px;
    margin-bottom: var(--spacing);
  }
  .header h1 {
    font-size: 22pt;
    font-weight: 700;
    margin-bottom: 6px;
  }
  .contact-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 16px;
    font-size: 9pt;
    opacity: 0.9;
  }
  .contact-row a { color: var(--primary-contrast); text-decoration: none; }

  /* Section with colored strip */
  .section { margin-bottom: var(--spacing); }
  .section-title {
    font-size: 12pt;
    font-weight: 700;
    color: var(--primary-contrast);
    background: var(--primary);
    padding: 4px 10px;
    border-radius: 3px;
    margin-bottom: calc(var(--spacing) * 0.5);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }

  /* Entries */
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #777; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }

  /* Pill skills */
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag {
    display: inline-block; background: var(--primary); color: var(--primary-contrast);
    padding: 2px 10px; border-radius: 12px; font-size: 9pt;
  }
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

// accentTemplate is a clean layout with colored left-border accent bars
// on the name and tinted section headings.
const accentTemplate = `<!DOCTYPE html>
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
    --text-color: {{if .TextColor}}{{.TextColor}}{{else}}#2d2d2d{{end}};
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: var(--text-color);
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }
  .container { padding: var(--spacing); }

  /* Header with left accent border */
  .header {
    border-left: 6px solid var(--primary);
    padding-left: calc(var(--spacing) * 0.8);
    margin-bottom: var(--spacing);
  }
  .header h1 {
    font-size: 22pt;
    font-weight: 700;
    color: var(--text-color);
    margin-bottom: 6px;
  }
  .contact-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 16px;
    font-size: 9pt;
    color: #555;
    padding-top: 6px;
    border-top: 1px solid #e0e0e0;
  }
  .contact-row a { color: #555; text-decoration: none; }

  /* Sections */
  .section { margin-bottom: var(--spacing); }
  .section-title {
    font-size: 12pt;
    font-weight: 700;
    color: var(--primary);
    padding-bottom: 4px;
    border-bottom: 2px solid var(--primary);
    margin-bottom: calc(var(--spacing) * 0.5);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }

  /* Entries */
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #777; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }

  /* Skills as text with levels */
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
    <div class="section-title">Professional Summary</div>
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

// timelineTemplate is a chronological layout with a decorative colored line
// under the name and clean section headers with pill-style skills.
const timelineTemplate = `<!DOCTYPE html>
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
    --text-color: {{if .TextColor}}{{.TextColor}}{{else}}#2d2d2d{{end}};
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: var(--font);
    font-size: 10pt;
    line-height: 1.5;
    color: var(--text-color);
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
  }
  .container { padding: var(--spacing); }

  /* Header with colored underline */
  .header {
    margin-bottom: var(--spacing);
  }
  .header h1 {
    font-size: 22pt;
    font-weight: 700;
    color: var(--text-color);
    margin-bottom: 6px;
  }
  .header-line {
    height: 3px;
    background: var(--primary);
    border-radius: 2px;
    margin-bottom: 8px;
  }
  .contact-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 16px;
    font-size: 9pt;
    color: #555;
  }
  .contact-row a { color: #555; text-decoration: none; }

  /* Sections */
  .section { margin-bottom: var(--spacing); }
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
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }

  /* Timeline entries with left border and dot */
  .timeline-entry {
    margin-bottom: calc(var(--spacing) * 0.6);
    padding-left: 16px;
    border-left: 2px solid var(--primary);
    position: relative;
    page-break-inside: avoid;
  }
  .timeline-entry::before {
    content: '';
    position: absolute;
    left: -5px;
    top: 4px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--primary);
  }
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #777; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }

  /* Pill skills */
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag {
    display: inline-block; background: var(--primary); color: #fff;
    padding: 2px 10px; border-radius: 12px; font-size: 9pt;
  }
  .inline-list { font-size: 10pt; line-height: 1.7; color: var(--text-color); }
  .gpa { font-size: 9pt; color: #666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
<div class="container">

  {{if .Contact}}
  <div class="header">
    <h1>{{.Contact.FullName}}</h1>
    <div class="header-line"></div>
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
    <div class="timeline-entry">
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
    <div class="timeline-entry">
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
    <div class="timeline-entry">
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
    <div class="timeline-entry">
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
    <div class="timeline-entry">
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

// vividTemplate is a colorful two-tone layout with a colored top header section
// and a white contact row with icon badges, pill-style skills.
const vividTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{if .Contact}}{{.Contact.FullName}}{{else}}Resume{{end}}</title>
<style>
  :root {
    --primary: {{.PrimaryColor}};
    --primary-contrast: {{contrastColor .PrimaryColor}};
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

  /* Two-tone header */
  .header-wrapper {
    border-radius: 8px;
    overflow: hidden;
    margin-bottom: var(--spacing);
  }
  .header-top {
    background: var(--primary);
    color: var(--primary-contrast);
    padding: calc(var(--spacing) * 1.2) var(--spacing);
  }
  .header-top h1 {
    font-size: 22pt;
    font-weight: 700;
  }
  .header-bottom {
    background: #ffffff;
    border: 1px solid #e0e0e0;
    border-top: none;
    border-radius: 0 0 8px 8px;
    padding: 8px var(--spacing);
  }
  .contact-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 16px;
    font-size: 9pt;
    color: #555;
  }
  .contact-row a { color: #555; text-decoration: none; }
  .contact-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
  .contact-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--primary);
    color: var(--primary-contrast);
    font-size: 8px;
    font-weight: 700;
  }

  /* Sections */
  .section { margin-bottom: var(--spacing); }
  .section-title {
    font-size: 12pt;
    font-weight: 700;
    color: var(--primary);
    padding-bottom: 4px;
    border-bottom: 2px solid var(--primary);
    margin-bottom: calc(var(--spacing) * 0.5);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }
  .summary-text { font-size: 10pt; line-height: 1.6; color: #444; }

  /* Entries */
  .entry { margin-bottom: calc(var(--spacing) * 0.6); page-break-inside: avoid; }
  .entry-header { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 2px; }
  .entry-title { font-weight: 700; font-size: 10.5pt; color: #1a1a1a; }
  .entry-date { font-size: 9pt; color: #777; white-space: nowrap; margin-left: 12px; }
  .entry-subtitle { font-size: 9.5pt; color: #555; margin-bottom: 3px; }
  .entry-description { font-size: 9.5pt; line-height: 1.5; color: #444; }

  /* Pill skills */
  .skill-tags { display: flex; flex-wrap: wrap; gap: 6px; }
  .skill-tag {
    display: inline-block; background: var(--primary); color: var(--primary-contrast);
    padding: 2px 10px; border-radius: 12px; font-size: 9pt;
  }
  .inline-list { font-size: 10pt; line-height: 1.7; color: #444; }
  .gpa { font-size: 9pt; color: #666; }
  .cert-link { color: var(--primary); text-decoration: none; font-size: 9pt; }
</style>
</head>
<body>
<div class="container">

  {{if .Contact}}
  <div class="header-wrapper">
    <div class="header-top">
      <h1>{{.Contact.FullName}}</h1>
    </div>
    <div class="header-bottom">
      <div class="contact-row">
        {{if .Contact.Email}}<span class="contact-badge"><span class="contact-icon">@</span> {{.Contact.Email}}</span>{{end}}
        {{if .Contact.Phone}}<span class="contact-badge"><span class="contact-icon">&#9742;</span> {{.Contact.Phone}}</span>{{end}}
        {{if .Contact.Location}}<span class="contact-badge"><span class="contact-icon">&#9679;</span> {{.Contact.Location}}</span>{{end}}
        {{if .Contact.Website}}{{with safeURL .Contact.Website}}<span class="contact-badge"><span class="contact-icon">&#9741;</span> <a href="{{.}}">{{$.Contact.Website}}</a></span>{{end}}{{end}}
        {{if .Contact.LinkedIn}}{{with safeURL .Contact.LinkedIn}}<span class="contact-badge"><span class="contact-icon">in</span> <a href="{{.}}">LinkedIn</a></span>{{end}}{{end}}
        {{if .Contact.GitHub}}{{with safeURL .Contact.GitHub}}<span class="contact-badge"><span class="contact-icon">&lt;/&gt;</span> <a href="{{.}}">GitHub</a></span>{{end}}{{end}}
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
