package micro

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"

	"github.com/resend/resend-go/v2"
	log "github.com/sirupsen/logrus"
)

type EmailSender interface {
	Send(msg EmailMessage) error
}

type Attachment struct {
	Filename    string
	Data        []byte
	ContentType string
}

type EmailMessage struct {
	To           string
	Cc           string
	Subject      string
	Template     string
	TemplateData map[string]any
	Html         string
	Text         string
	Attachments  []Attachment
}

type EmailTemplate struct {
	Html *template.Template
	Text *template.Template
}

type mailProviderConfig struct {
	Scheme      string            // "mail"
	APIKey      string            // "apikey"
	APIPassword string            // "apipassword" (optional)
	Host        string            // "resend"
	QueryParams map[string]string // all query parameters
}

func ConfigureEmailProvider(provider string) (EmailSender, error) {
	cfg, err := parseProviderUrl(provider)
	if err != nil {
		return nil, err
	}
	appName := cfg.QueryParams["appName"]
	sender := cfg.QueryParams["sender"]
	if sender == "" {
		return nil, fmt.Errorf("[email-provider] sender is required")
	}

	fullSender := fmt.Sprintf("%s <%s>", appName, sender)
	switch cfg.Host {
	case "resend":
		return newResendEmailProvider(cfg.APIKey, fullSender), nil
	case "faker":
		return newFakeEmailProvider("", fullSender), nil
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Host)
	}

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

/// ------------------------------------------------------------
/// ------------------------------------------------------------

type ResendEmailProvider struct {
	EmailSender
	client *resend.Client
	sender string
}

func newResendEmailProvider(apikey string, sender string) EmailSender {
	client := resend.NewClient(apikey)
	return &ResendEmailProvider{
		client: client,
		sender: sender,
	}
}

func (p *ResendEmailProvider) Send(msg EmailMessage) error {
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
		log.Errorf("failed to send email: %v", err)
		return err
	}
	log.Infof("email %s send to %s", msg.Subject, msg.To)
	return nil
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

type FakeEmailProvider struct {
	EmailSender
	sender string
}

func newFakeEmailProvider(_ string, sender string) EmailSender {
	return &FakeEmailProvider{
		sender: sender,
	}
}

func (p *FakeEmailProvider) Send(msg EmailMessage) error {
	err := parseEmailMessage(&msg)
	if err != nil {
		return err
	}
	log.Infof("email %s send to %s", msg.Subject, msg.To)
	log.Infof("--------------------------------")
	log.Infof("%s", msg.Html)
	log.Infof("--------------------------------")
	return nil
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

func parseEmailMessage(templateDir fs.FS, msg *EmailMessage) error {
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
