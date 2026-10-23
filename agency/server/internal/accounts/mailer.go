// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package accounts

import (
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/audit"
)

// SMTP settings keys (the `settings` key/value table).
const (
	settingSMTPHost       = "smtp.host"
	settingSMTPPort       = "smtp.port"
	settingSMTPUsername   = "smtp.username"
	settingSMTPPassword   = "smtp.password"
	settingSMTPFrom       = "smtp.from"
	settingSMTPEncryption = "smtp.encryption"
)

// SMTPSettings is the org-wide outbound mail configuration. The password is
// write-only: reads leave it blank and report whether one is stored.
type SMTPSettings struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	Encryption string // "none", "starttls" or "tls"
}

// Configured reports whether enough is set to attempt delivery.
func (s SMTPSettings) Configured() bool {
	return strings.TrimSpace(s.Host) != "" && strings.TrimSpace(s.From) != ""
}

// MailMessage is one outbound plain-text message.
type MailMessage struct {
	To      string
	Subject string
	Text    string
}

// Mailer delivers a message through the configured SMTP server.
type Mailer interface {
	Send(ctx context.Context, settings SMTPSettings, msg MailMessage) error
}

// SMTPSender is the standard-library mailer (plain, STARTTLS or implicit TLS).
type SMTPSender struct{}

// GetSMTPSettings reads the mail configuration. The stored password is
// deliberately not returned; callers that need it use smtpPassword.
func (s *Service) GetSMTPSettings(ctx context.Context) (SMTPSettings, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return SMTPSettings{}, err
	}
	out := SMTPSettings{Encryption: "starttls"}
	for _, row := range rows {
		switch row.Key {
		case settingSMTPHost:
			out.Host = row.Value
		case settingSMTPPort:
			if value, err := strconv.Atoi(strings.TrimSpace(row.Value)); err == nil {
				out.Port = value
			}
		case settingSMTPUsername:
			out.Username = row.Value
		case settingSMTPPassword:
			out.Password = row.Value
		case settingSMTPFrom:
			out.From = row.Value
		case settingSMTPEncryption:
			if row.Value != "" {
				out.Encryption = row.Value
			}
		}
	}
	return out, nil
}

// smtpPassword reads the write-only password for a send attempt.
func (s *Service) smtpPassword(ctx context.Context, settings SMTPSettings) SMTPSettings {
	if settings.Password != "" {
		return settings
	}
	value, err := s.store.GetSetting(ctx, settingSMTPPassword)
	if err == nil {
		settings.Password = value
	}
	return settings
}

// UpdateSMTPSettings persists the mail configuration. An empty password keeps
// the stored one, so the settings form can round-trip without re-typing it.
func (s *Service) UpdateSMTPSettings(ctx context.Context, in SMTPSettings) error {
	if strings.TrimSpace(in.Host) != "" && strings.TrimSpace(in.From) == "" {
		return ErrSMTPNotConfigured
	}
	if in.Encryption == "" {
		in.Encryption = "starttls"
	}
	switch in.Encryption {
	case "none", "starttls", "tls":
	default:
		return errors.New("invalid SMTP encryption")
	}
	values := []struct{ key, value string }{
		{settingSMTPHost, strings.TrimSpace(in.Host)},
		{settingSMTPPort, strconv.Itoa(in.Port)},
		{settingSMTPUsername, strings.TrimSpace(in.Username)},
		{settingSMTPFrom, strings.TrimSpace(in.From)},
		{settingSMTPEncryption, in.Encryption},
	}
	for _, item := range values {
		if err := s.store.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   item.key,
			Value: item.value,
		}); err != nil {
			return err
		}
	}
	if in.Password != "" {
		if err := s.store.UpsertSetting(ctx, store.UpsertSettingParams{
			Key:   settingSMTPPassword,
			Value: in.Password,
		}); err != nil {
			return err
		}
	}
	if actor := audit.ActorID(ctx); actor != 0 {
		s.auditUser(ctx, actor, audit.KindSMTPSettings, "", "", "",
			"Updated SMTP settings", nil)
	}
	return nil
}

// SMTPConfigured reports whether outbound mail can be attempted.
func (s *Service) SMTPConfigured(ctx context.Context) bool {
	settings, err := s.GetSMTPSettings(ctx)
	return err == nil && settings.Configured()
}

// sendMail delivers one message if SMTP is configured. A missing configuration
// is a no-op, so copyable links still work on SMTP-less installs.
func (s *Service) sendMail(ctx context.Context, to, subject, text string) error {
	if strings.TrimSpace(to) == "" {
		return ErrSMTPNotConfigured
	}
	settings, err := s.GetSMTPSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Configured() {
		return ErrSMTPNotConfigured
	}
	return s.mailer.Send(ctx, s.smtpPassword(ctx, settings), MailMessage{
		To:      to,
		Subject: subject,
		Text:    text,
	})
}

// Send implements Mailer over net/smtp.
func (SMTPSender) Send(
	ctx context.Context,
	settings SMTPSettings,
	msg MailMessage,
) error {
	if !settings.Configured() {
		return ErrSMTPNotConfigured
	}
	host := strings.TrimSpace(settings.Host)
	port := settings.Port
	if port == 0 {
		switch settings.Encryption {
		case "tls":
			port = 465
		case "starttls":
			port = 587
		default:
			port = 25
		}
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if settings.Encryption == "tls" {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if settings.Encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("accounts: SMTP server does not offer STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if settings.Username != "" {
		auth := smtp.PlainAuth("", settings.Username, settings.Password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(settings.From); err != nil {
		return err
	}
	if err := client.Rcpt(msg.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(formatMail(settings.From, msg))); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// formatMail renders a minimal RFC 5322 message.
func formatMail(from string, msg MailMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@cherrypi>\r\n", randomToken(16))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(msg.Text, "\r\n", "\n"), "\n", "\r\n"))
	b.WriteString("\r\n")
	return b.String()
}
