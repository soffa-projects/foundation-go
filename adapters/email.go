package adapters

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/resend/resend-go/v2"
	core "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/log"
)

type ResendEmailProvider struct {
	core.EmailSender
	client *resend.Client
	sender string
}

type mailProviderConfig struct {
	Scheme      string            // "mail"
	APIKey      string            // "apikey"
	APIPassword string            // "apipassword" (optional)
	Host        string            // "resend"
	QueryParams map[string]string // all query parameters
}

func newResendEmailProvider(apikey string, sender string) core.EmailSender {
	client := resend.NewClient(apikey)
	return &ResendEmailProvider{
		client: client,
		sender: sender,
	}
}

func (p *ResendEmailProvider) Send(msg core.EmailMessage) error {
	ctx := context.Background()
	err := parseEmailMessage(&msg)
	if err != nil {
		return err
	}
	params := &resend.SendEmailRequest{
		From:    p.sender,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.Html,
	}
	if msg.Cc != "" {
		params.Cc = []string{msg.Cc}
	}
	if len(msg.Attachments) > 0 {
		params.Attachments = make([]*resend.Attachment, len(msg.Attachments))

		for i, attachment := range msg.Attachments {
			params.Attachments[i] = &resend.Attachment{
				Filename:    attachment.Filename,
				Content:     attachment.Data,
				ContentType: attachment.ContentType,
			}
		}
	}
	_, err = p.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Error("failed to send email: %v", err)
		return err
	}
	log.Info("email %s send to %s", msg.Subject, msg.To)
	return nil
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

type FakeEmailProvider struct {
	core.EmailSender
	sender string
}

func newFakeEmailProvider(_ string, sender string) core.EmailSender {
	return &FakeEmailProvider{
		sender: sender,
	}
}

func (p *FakeEmailProvider) Send(msg core.EmailMessage) error {
	err := parseEmailMessage(&msg)
	if err != nil {
		return err
	}
	log.Info("email %s send to %s", msg.Subject, msg.To)
	log.Info("--------------------------------")
	log.Info("%s", msg.Html)
	log.Info("--------------------------------")
	return nil
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

func parseEmailMessage(msg *core.EmailMessage) error {
	templateDir := msg.TemplateDir
	if msg.Template != "" {
		tpl, err := template.ParseFS(templateDir, fmt.Sprintf("templates/emails/%s.html", msg.Template))
		if err != nil {
			return err
		}
		var htmlContent bytes.Buffer
		err = tpl.Execute(&htmlContent, msg.TemplateData)
		if err != nil {
			return err
		}
		msg.Html = htmlContent.String()
	}
	return nil
}

func NewEmailSender(provider string) core.EmailSender {
	cfg, err := parseProviderUrl(provider)
	if err != nil {
		log.Fatal("failed to parse email provider: %v", err)
	}
	appName := cfg.QueryParams["appName"]
	sender := cfg.QueryParams["sender"]
	if sender == "" {
		log.Fatal("sender is required")
	}

	fullSender := fmt.Sprintf("%s <%s>", appName, sender)
	switch cfg.Host {
	case "resend":
		return newResendEmailProvider(cfg.APIKey, fullSender)
	case "faker":
		return newFakeEmailProvider("", fullSender)
	}
	log.Fatal("unsupported email provider: %s", cfg.Host)
	return nil
}

func parseProviderUrl(provider string) (*mailProviderConfig, error) {
	// Parse the URL
	u, err := url.Parse(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email provider: %w", err)
	}

	cfg := &mailProviderConfig{
		Scheme:      u.Scheme,
		Host:        u.Host,
		QueryParams: make(map[string]string),
	}

	// Extract API key and password from user info
	if u.User != nil {
		cfg.APIKey = u.User.Username()
		if password, hasPassword := u.User.Password(); hasPassword {
			cfg.APIPassword = password
		}
	}

	// Parse query parameters into map
	for key, values := range u.Query() {
		if len(values) > 0 {
			cfg.QueryParams[key] = values[0] // Take first value if multiple
		}
	}

	return cfg, nil
}
