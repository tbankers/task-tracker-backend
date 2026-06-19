package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

func sendEmail(to, subject, body string) error {
	if smtpCfg.Username == "" || smtpCfg.Password == "" {
		fmt.Printf("[EMAIL STUB] To: %s | Subject: %s\n", to, subject)
		return nil
	}

	headers := make(map[string]string)
	headers["From"] = smtpCfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	addr := net.JoinHostPort(smtpCfg.Host, fmt.Sprintf("%d", smtpCfg.Port))

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("net dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, smtpCfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err = client.StartTLS(&tls.Config{ServerName: smtpCfg.Host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(smtpCfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	return client.Quit()
}

func escHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

const emailBaseTemplate = `<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#F1F5F9;font-family:Inter,-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#F1F5F9;padding:40px 20px;">
<tr><td align="center">
<table width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,.08);">
{{.Content}}
</table>
<p style="color:#94A3B8;font-size:12px;margin-top:20px;">Task Tracker &middot; Управление задачами</p>
</td></tr></table>
</body></html>`

const emailHeaderBlock = `<tr><td style="background:linear-gradient(135deg,#1E1B4B,#312E81,#4C1D95);padding:32px 40px;text-align:center;">
<h1 style="color:#fff;font-size:22px;margin:0;font-weight:700;">Task<span style="color:#8B5CF6">Tracker</span></h1>
</td></tr>`

func renderEmail(content string) string {
	tmpl := emailBaseTemplate
	tmpl = strings.Replace(tmpl, "{{.Content}}", emailHeaderBlock+content, 1)
	return tmpl
}

func verificationEmailHTML(username, link string) string {
	content := fmt.Sprintf(`<tr><td style="padding:40px;">
<h2 style="color:#1E293B;font-size:20px;margin:0 0 8px;">Подтвердите email</h2>
<p style="color:#64748B;font-size:14px;line-height:1.6;margin:0 0 24px;">Привет, %s! Завершите регистрацию, нажав кнопку ниже.</p>
<table cellpadding="0" cellspacing="0" style="margin:0 auto;"><tr><td style="background:linear-gradient(135deg,#7C3AED,#4338CA);border-radius:8px;">
<a href="%s" style="display:inline-block;padding:14px 32px;color:#fff;font-size:15px;font-weight:600;text-decoration:none;">Подтвердить email</a>
</td></tr></table>
<p style="color:#94A3B8;font-size:12px;margin:28px 0 0;line-height:1.6;">Если вы не регистрировались в Task Tracker, просто проигнорируйте это письмо.</p>
</td></tr>`, escHTML(username), link)
	return renderEmail(content)
}

func passwordResetEmailHTML(email, link string) string {
	content := fmt.Sprintf(`<tr><td style="padding:40px;">
<h2 style="color:#1E293B;font-size:20px;margin:0 0 8px;">Сброс пароля</h2>
<p style="color:#64748B;font-size:14px;line-height:1.6;margin:0 0 24px;">Мы получили запрос на сброс пароля для аккаунта <strong>%s</strong>. Нажмите кнопку ниже, чтобы задать новый пароль.</p>
<table cellpadding="0" cellspacing="0" style="margin:0 auto;"><tr><td style="background:linear-gradient(135deg,#7C3AED,#4338CA);border-radius:8px;">
<a href="%s" style="display:inline-block;padding:14px 32px;color:#fff;font-size:15px;font-weight:600;text-decoration:none;">Сбросить пароль</a>
</td></tr></table>
<p style="color:#94A3B8;font-size:12px;margin:28px 0 0;line-height:1.6;">Ссылка действительна в течение 1 часа. Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>
</td></tr>`, escHTML(email), link)
	return renderEmail(content)
}

func passwordChangedEmailHTML(email string) string {
	content := fmt.Sprintf(`<tr><td style="padding:40px;">
<h2 style="color:#1E293B;font-size:20px;margin:0 0 8px;">Пароль изменён</h2>
<p style="color:#64748B;font-size:14px;line-height:1.6;margin:0 0 24px;">Пароль для аккаунта <strong>%s</strong> был успешно изменён.</p>
<p style="color:#94A3B8;font-size:12px;margin:28px 0 0;line-height:1.6;">Если вы не меняли пароль, немедленно обратитесь в поддержку.</p>
</td></tr>`, escHTML(email))
	return renderEmail(content)
}
