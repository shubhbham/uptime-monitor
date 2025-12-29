package email

import (
	"context"
	"fmt"
	"log"
	"time"

	brevo "github.com/getbrevo/brevo-go/lib"
	"github.com/shubhbham/uptime-monitor/internal/config"
)

type Service struct {
	client *brevo.APIClient
	cfg    *config.EmailConfig
}

func NewService(cfg *config.EmailConfig) *Service {
	brevoCfg := brevo.NewConfiguration()
	brevoCfg.AddDefaultHeader("api-key", cfg.BrevoAPIKey)
	// BasePath is handled by the SDK default (https://api.brevo.com/v3)

	client := brevo.NewAPIClient(brevoCfg)

	return &Service{
		client: client,
		cfg:    cfg,
	}
}

func (s *Service) SendIncidentOpened(ctx context.Context, monitorName, monitorURL, cause string, userEmail string) error {
	if !s.cfg.AlertsEnabled || !s.cfg.AlertOnIncidentOpen {
		return nil
	}

	if userEmail == "" {
		log.Printf("Cannot send incident email for %s: user email not found", monitorName)
		return nil
	}

	subject := fmt.Sprintf("🔴 [Incident] %s is DOWN", monitorName)
	htmlContent := fmt.Sprintf(`
		<h2>Monitor Down: %s</h2>
		<p>The monitor <strong>%s</strong> (%s) is currently down.</p>
		<p><strong>Cause:</strong> %s</p>
		<p>Time: %s</p>
		<p>--<br>Uptime Monitor</p>
	`, monitorName, monitorName, monitorURL, cause, time.Now().Format(time.RFC1123))

	return s.sendWithRetry(ctx, subject, htmlContent, userEmail)
}

func (s *Service) SendIncidentResolved(ctx context.Context, monitorName, monitorURL string, durationStr string, userEmail string) error {
	if !s.cfg.AlertsEnabled || !s.cfg.AlertOnIncidentResolve {
		return nil
	}

	if userEmail == "" {
		log.Printf("Cannot send resolved email for %s: user email not found", monitorName)
		return nil
	}

	subject := fmt.Sprintf("🟢 [Resolved] %s is UP", monitorName)
	htmlContent := fmt.Sprintf(`
		<h2>Monitor Resolved: %s</h2>
		<p>The monitor <strong>%s</strong> (%s) is back up.</p>
		<p><strong>Downtime Duration:</strong> %s</p>
		<p>--<br>Uptime Monitor</p>
	`, monitorName, monitorName, monitorURL, durationStr)

	return s.sendWithRetry(ctx, subject, htmlContent, userEmail)
}

func (s *Service) sendWithRetry(ctx context.Context, subject, htmlContent, toEmail string) error {
	var err error
	maxRetries := s.cfg.AlertMaxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	for i := 0; i < maxRetries; i++ {
		err = s.send(ctx, subject, htmlContent, toEmail)
		if err == nil {
			return nil
		}

		log.Printf("Failed to send email (attempt %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(s.cfg.AlertRetryBackoffSeconds) * time.Second):
				continue
			}
		}
	}
	return fmt.Errorf("failed to send email after %d attempts: %w", maxRetries, err)
}

func (s *Service) send(ctx context.Context, subject, htmlContent, toEmail string) error {
	if s.cfg.BrevoSandboxMode {
		log.Printf("[SANDBOX] Would send email: Subject='%s' To='%s'", subject, toEmail)
		return nil
	}

	sender := &brevo.SendSmtpEmailSender{
		Name:  s.cfg.AlertFromName,
		Email: s.cfg.AlertFromEmail,
	}

	replyTo := &brevo.SendSmtpEmailReplyTo{
		Email: s.cfg.AlertReplyToEmail,
		Name:  s.cfg.AlertReplyToName,
	}

	to := []brevo.SendSmtpEmailTo{
		{
			Email: toEmail,
			Name:  "User", // Could pass user name if available
		},
	}

	tags := []string{s.cfg.AlertTagService, s.cfg.AlertTagIncident}

	email := brevo.SendSmtpEmail{
		Sender:      sender,
		ReplyTo:     replyTo,
		To:          to,
		Subject:     subject,
		HtmlContent: htmlContent,
		Tags:        tags,
	}

	_, resp, err := s.client.TransactionalEmailsApi.SendTransacEmail(ctx, email)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("brevo api returned non-2xx status: %d", resp.StatusCode)
	}

	log.Printf("Sent email for %s to %s", subject, toEmail)
	return nil
}
