package pdf

// clEmbeddedTemplates holds the HTML templates for cover letter PDF generation.
// Prefixed with "cl_" to avoid collisions with resume templates.
var clEmbeddedTemplates = map[string]string{
	"professional": clProfessionalTemplate,
	"modern":       clModernTemplate,
	"minimal":      clMinimalTemplate,
	"executive":    clExecutiveTemplate,
	"creative":     clCreativeTemplate,
}

// clProfessionalTemplate — header with bottom accent border, clean corporate look.
const clProfessionalTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: {{.FontFamily}}, sans-serif;
    font-size: {{.FontSize}}px;
    line-height: 1.6;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
    padding: 40px;
  }
  .header {
    padding-bottom: 16px;
    margin-bottom: 24px;
    border-bottom: 2px solid {{.PrimaryColor}};
  }
  .header p { font-size: 10pt; color: #374151; margin-bottom: 2px; }
  .header .company { font-weight: 600; color: #1f2937; }
  .header .address { color: #6b7280; white-space: pre-line; }
  .date { font-size: 10pt; color: #6b7280; margin-bottom: 16px; }
  .greeting { font-weight: 600; color: #1f2937; margin-bottom: 16px; }
  .paragraph { color: #374151; margin-bottom: 12px; line-height: 1.7; }
  .closing { margin-top: 24px; font-weight: 600; color: #1f2937; }
</style>
</head>
<body>
  <div class="header">
    {{if .RecipientName}}<p>{{.RecipientName}}</p>{{end}}
    {{if .RecipientTitle}}<p style="color:#6b7280">{{.RecipientTitle}}</p>{{end}}
    {{if .CompanyName}}<p class="company">{{.CompanyName}}</p>{{end}}
    {{if .CompanyAddress}}<p class="address">{{.CompanyAddress}}</p>{{end}}
  </div>
  <p class="date">{{.Date}}</p>
  {{if .Greeting}}<p class="greeting">{{.Greeting}}</p>{{end}}
  {{range .Paragraphs}}<p class="paragraph">{{if .}}{{.}}{{else}}&nbsp;{{end}}</p>{{end}}
  {{if .Closing}}<div class="closing">{{.Closing}}</div>{{end}}
</body>
</html>`

// clModernTemplate — left colored sidebar bar next to header, accent-colored greeting/closing.
const clModernTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: {{.FontFamily}}, sans-serif;
    font-size: {{.FontSize}}px;
    line-height: 1.6;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
    padding: 40px;
  }
  .header {
    display: flex;
    gap: 16px;
    margin-bottom: 24px;
  }
  .header-bar {
    width: 4px;
    flex-shrink: 0;
    border-radius: 4px;
    background-color: {{.PrimaryColor}};
  }
  .header-content p { font-size: 10pt; margin-bottom: 2px; }
  .header-content .company { font-size: 12pt; font-weight: 700; color: {{.PrimaryColor}}; }
  .header-content .name { color: #374151; }
  .header-content .title { color: #6b7280; }
  .header-content .address { color: #6b7280; white-space: pre-line; }
  .date { font-size: 10pt; color: #9ca3af; margin-bottom: 16px; }
  .greeting { color: {{.PrimaryColor}}; margin-bottom: 16px; }
  .paragraph { color: #374151; margin-bottom: 12px; line-height: 1.7; }
  .closing { margin-top: 24px; color: {{.PrimaryColor}}; }
</style>
</head>
<body>
  <div class="header">
    <div class="header-bar"></div>
    <div class="header-content">
      {{if .CompanyName}}<p class="company">{{.CompanyName}}</p>{{end}}
      {{if .RecipientName}}<p class="name">{{.RecipientName}}</p>{{end}}
      {{if .RecipientTitle}}<p class="title">{{.RecipientTitle}}</p>{{end}}
      {{if .CompanyAddress}}<p class="address">{{.CompanyAddress}}</p>{{end}}
    </div>
  </div>
  <p class="date">{{.Date}}</p>
  {{if .Greeting}}<p class="greeting">{{.Greeting}}</p>{{end}}
  {{range .Paragraphs}}<p class="paragraph">{{if .}}{{.}}{{else}}&nbsp;{{end}}</p>{{end}}
  {{if .Closing}}<div class="closing">{{.Closing}}</div>{{end}}
</body>
</html>`

// clMinimalTemplate — no accent colors, ultra-clean typography.
const clMinimalTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: {{.FontFamily}}, sans-serif;
    font-size: {{.FontSize}}px;
    line-height: 1.6;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
    padding: 40px;
  }
  .header { margin-bottom: 24px; }
  .header p { font-size: 10pt; margin-bottom: 2px; }
  .header .name { color: #1f2937; }
  .header .title { color: #6b7280; }
  .header .company { color: #6b7280; }
  .header .address { color: #9ca3af; white-space: pre-line; }
  .date { font-size: 10pt; color: #9ca3af; margin-bottom: 24px; }
  .greeting { color: #1f2937; margin-bottom: 16px; }
  .paragraph { color: #374151; margin-bottom: 12px; line-height: 1.7; }
  .closing { margin-top: 32px; color: #1f2937; }
</style>
</head>
<body>
  <div class="header">
    {{if .RecipientName}}<p class="name">{{.RecipientName}}</p>{{end}}
    {{if .RecipientTitle}}<p class="title">{{.RecipientTitle}}</p>{{end}}
    {{if .CompanyName}}<p class="company">{{.CompanyName}}</p>{{end}}
    {{if .CompanyAddress}}<p class="address">{{.CompanyAddress}}</p>{{end}}
  </div>
  <p class="date">{{.Date}}</p>
  {{if .Greeting}}<p class="greeting">{{.Greeting}}</p>{{end}}
  {{range .Paragraphs}}<p class="paragraph">{{if .}}{{.}}{{else}}&nbsp;{{end}}</p>{{end}}
  {{if .Closing}}<div class="closing">{{.Closing}}</div>{{end}}
</body>
</html>`

// clExecutiveTemplate — bold colored header block with white text.
const clExecutiveTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: {{.FontFamily}}, sans-serif;
    font-size: {{.FontSize}}px;
    line-height: 1.6;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
    padding: 40px;
  }
  .header {
    background-color: {{.PrimaryColor}};
    border-radius: 8px;
    padding: 20px 24px;
    margin-bottom: 24px;
  }
  .header p { margin-bottom: 2px; }
  .header .name { font-size: 12pt; font-weight: 700; color: #ffffff; }
  .header .title { font-size: 10pt; color: rgba(255,255,255,0.8); }
  .header .company { font-size: 10pt; font-weight: 600; color: rgba(255,255,255,0.9); }
  .header .address { font-size: 10pt; color: rgba(255,255,255,0.7); white-space: pre-line; }
  .date { font-size: 10pt; color: #6b7280; margin-bottom: 16px; }
  .greeting { font-weight: 700; color: #1f2937; margin-bottom: 16px; }
  .paragraph { color: #374151; margin-bottom: 12px; line-height: 1.7; }
  .closing { margin-top: 24px; font-weight: 700; color: #1f2937; }
</style>
</head>
<body>
  <div class="header">
    {{if .RecipientName}}<p class="name">{{.RecipientName}}</p>{{end}}
    {{if .RecipientTitle}}<p class="title">{{.RecipientTitle}}</p>{{end}}
    {{if .CompanyName}}<p class="company">{{.CompanyName}}</p>{{end}}
    {{if .CompanyAddress}}<p class="address">{{.CompanyAddress}}</p>{{end}}
  </div>
  <p class="date">{{.Date}}</p>
  {{if .Greeting}}<p class="greeting">{{.Greeting}}</p>{{end}}
  {{range .Paragraphs}}<p class="paragraph">{{if .}}{{.}}{{else}}&nbsp;{{end}}</p>{{end}}
  {{if .Closing}}<div class="closing">{{.Closing}}</div>{{end}}
</body>
</html>`

// clCreativeTemplate — left colored sidebar panel with white text, content on the right.
const clCreativeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: {{.FontFamily}}, sans-serif;
    font-size: {{.FontSize}}px;
    line-height: 1.6;
    color: #2d2d2d;
    width: 210mm;
    min-height: 297mm;
    background: #ffffff;
    padding: 40px;
  }
  .layout { display: flex; gap: 24px; }
  .sidebar {
    width: 160px;
    flex-shrink: 0;
    background-color: {{.PrimaryColor}};
    border-radius: 8px;
    padding: 20px 16px;
  }
  .sidebar p { margin-bottom: 4px; }
  .sidebar .name { font-size: 10pt; font-weight: 700; color: #ffffff; margin-bottom: 4px; }
  .sidebar .title { font-size: 9pt; color: rgba(255,255,255,0.8); margin-bottom: 12px; }
  .sidebar .company { font-size: 9pt; font-weight: 600; color: rgba(255,255,255,0.9); margin-bottom: 4px; }
  .sidebar .address { font-size: 9pt; color: rgba(255,255,255,0.7); white-space: pre-line; }
  .content { flex: 1; }
  .date { font-size: 10pt; color: #9ca3af; margin-bottom: 16px; }
  .greeting { font-weight: 600; color: {{.PrimaryColor}}; margin-bottom: 16px; }
  .paragraph { color: #374151; margin-bottom: 12px; line-height: 1.7; }
  .closing { margin-top: 24px; font-weight: 600; color: {{.PrimaryColor}}; }
</style>
</head>
<body>
  <div class="layout">
    <div class="sidebar">
      {{if .RecipientName}}<p class="name">{{.RecipientName}}</p>{{end}}
      {{if .RecipientTitle}}<p class="title">{{.RecipientTitle}}</p>{{end}}
      {{if .CompanyName}}<p class="company">{{.CompanyName}}</p>{{end}}
      {{if .CompanyAddress}}<p class="address">{{.CompanyAddress}}</p>{{end}}
    </div>
    <div class="content">
      <p class="date">{{.Date}}</p>
      {{if .Greeting}}<p class="greeting">{{.Greeting}}</p>{{end}}
      {{range .Paragraphs}}<p class="paragraph">{{if .}}{{.}}{{else}}&nbsp;{{end}}</p>{{end}}
      {{if .Closing}}<div class="closing">{{.Closing}}</div>{{end}}
    </div>
  </div>
</body>
</html>`
