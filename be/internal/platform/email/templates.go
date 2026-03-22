package email

import (
	"fmt"
	"html"
)

type emailContent struct {
	Subject string
	HTML    string
}

// baseLayout wraps email body content in a responsive, table-based HTML layout
// compatible with all major email clients (Gmail, Outlook, Apple Mail, Yahoo).
//
// title is HTML-escaped internally. body MUST be trusted HTML constructed from
// hardcoded template strings — never pass user-controlled content as body directly.
func baseLayout(title, body string) string {
	safeTitle := html.EscapeString(title)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
<title>%s</title>
<style type="text/css">
body,table,td,a{-webkit-text-size-adjust:100%%;-ms-text-size-adjust:100%%}
table,td{mso-table-lspace:0;mso-table-rspace:0}
img{-ms-interpolation-mode:bicubic;border:0;height:auto;line-height:100%%;outline:none;text-decoration:none}
body{margin:0;padding:0;width:100%%!important;height:100%%!important}
</style>
</head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif">

<!-- Outer wrapper -->
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background-color:#f4f4f5">
<tr><td align="center" style="padding:32px 16px">

<!-- Container 600px -->
<table role="presentation" width="600" cellspacing="0" cellpadding="0" border="0" style="max-width:600px;width:100%%">

<!-- Header -->
<tr>
<td style="background-color:#1e293b;padding:24px 32px;border-radius:12px 12px 0 0">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
<tr>
<td style="font-size:22px;font-weight:700;color:#ffffff;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;letter-spacing:-0.5px">
Jobber
</td>
</tr>
</table>
</td>
</tr>

<!-- Accent bar -->
<tr>
<td style="background-color:#2563eb;height:4px;font-size:0;line-height:0">&nbsp;</td>
</tr>

<!-- Body -->
<tr>
<td style="background-color:#ffffff;padding:40px 32px">
%s
</td>
</tr>

<!-- Footer -->
<tr>
<td style="background-color:#f9fafb;padding:24px 32px;border-radius:0 0 12px 12px;border-top:1px solid #e5e7eb">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
<tr>
<td style="font-size:13px;color:#9ca3af;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;line-height:20px;text-align:center">
&copy; 2026 Jobber. All rights reserved.<br/>
This is an automated message. Please do not reply directly to this email.
</td>
</tr>
</table>
</td>
</tr>

</table>
<!-- /Container -->

</td></tr>
</table>
<!-- /Outer wrapper -->

</body>
</html>`, safeTitle, body)
}

// codeBlockHTML returns a large, monospaced code block for email templates.
func codeBlockHTML(code string) string {
	return fmt.Sprintf(`<table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:28px 0" width="100%%">
<tr>
<td align="center">
<div style="display:inline-block;padding:16px 32px;font-size:32px;font-weight:700;letter-spacing:8px;font-family:'Courier New',Courier,monospace;color:#1e293b;background-color:#f1f5f9;border:2px dashed #cbd5e1;border-radius:8px">
%s
</div>
</td>
</tr>
</table>`, html.EscapeString(code))
}

func verificationEmail(code, locale string) emailContent {
	switch locale {
	case "ru":
		return emailContent{
			Subject: "Подтверждение email — Jobber",
			HTML: baseLayout("Подтверждение email", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Подтвердите ваш email</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Введите этот код в приложении, чтобы подтвердить ваш email:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">Код действителен 10 минут. Если вы не регистрировались на Jobber, проигнорируйте это письмо.</p>`,
				codeBlockHTML(code),
			)),
		}
	case "ua":
		return emailContent{
			Subject: "Підтвердження email — Jobber",
			HTML: baseLayout("Підтвердження email", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Підтвердіть ваш email</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Введіть цей код у додатку, щоб підтвердити ваш email:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">Код дійсний 10 хвилин. Якщо ви не реєструвалися на Jobber, проігноруйте цей лист.</p>`,
				codeBlockHTML(code),
			)),
		}
	default:
		return emailContent{
			Subject: "Verify your email — Jobber",
			HTML: baseLayout("Verify your email", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Verify your email</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Enter this code in the app to verify your email address:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">This code expires in 10 minutes. If you didn't sign up for Jobber, please ignore this email.</p>`,
				codeBlockHTML(code),
			)),
		}
	}
}

func passwordResetEmail(code, locale string) emailContent {
	switch locale {
	case "ru":
		return emailContent{
			Subject: "Сброс пароля — Jobber",
			HTML: baseLayout("Сброс пароля", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Сброс пароля</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Введите этот код в приложении, чтобы сбросить пароль:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">Код действителен 10 минут. Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>`,
				codeBlockHTML(code),
			)),
		}
	case "ua":
		return emailContent{
			Subject: "Скидання пароля — Jobber",
			HTML: baseLayout("Скидання пароля", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Скидання пароля</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Введіть цей код у додатку, щоб скинути пароль:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">Код дійсний 10 хвилин. Якщо ви не запитували скидання пароля, проігноруйте цей лист.</p>`,
				codeBlockHTML(code),
			)),
		}
	default:
		return emailContent{
			Subject: "Reset your password — Jobber",
			HTML: baseLayout("Reset your password", fmt.Sprintf(
				`<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;color:#1e293b">Reset your password</h1>
<p style="margin:0 0 8px;font-size:16px;color:#475569;line-height:26px">Enter this code in the app to reset your password:</p>
%s
<p style="margin:0;font-size:14px;color:#94a3b8;line-height:22px">This code expires in 10 minutes. If you didn't request a password reset, please ignore this email.</p>`,
				codeBlockHTML(code),
			)),
		}
	}
}
