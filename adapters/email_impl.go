package adapters

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/resend/resend-go/v2"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/log"
)

type BaseEmailProvider struct {
	f.EmailSender
	hooks map[string]func(msg f.EmailMessage)
}

func (p *BaseEmailProvider) On(name string, fn func(msg f.EmailMessage)) {
	p.hooks[name] = fn
}

func (p *BaseEmailProvider) callHook(name string, msg f.EmailMessage) {
	fn, ok := p.hooks[name]
	if !ok {
		return
	}
	fn(msg)
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

type ResendEmailProvider struct {
	BaseEmailProvider
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

func newResendEmailProvider(apikey string, sender string) f.EmailSender {
	client := resend.NewClient(apikey)
	return &ResendEmailProvider{
		client: client,
		sender: sender,
		BaseEmailProvider: BaseEmailProvider{
			hooks: make(map[string]func(msg f.EmailMessage)),
		},
	}
}

func (p *ResendEmailProvider) Send(msg f.EmailMessage) error {
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
	p.callHook("send", msg)
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
	BaseEmailProvider
	sender string
}

func newFakeEmailProvider(_ string, sender string) f.EmailSender {
	return &FakeEmailProvider{
		sender: sender,
		BaseEmailProvider: BaseEmailProvider{
			hooks: make(map[string]func(msg f.EmailMessage)),
		},
	}
}

func (p *FakeEmailProvider) Send(msg f.EmailMessage) error {
	err := parseEmailMessage(&msg)
	if err != nil {
		return err
	}
	p.callHook("send", msg)
	log.Info("email %s send to %s", msg.Subject, msg.To)
	log.Info("--------------------------------")
	log.Info("%s", msg.Text)
	log.Info("--------------------------------")
	return nil
}

/// ------------------------------------------------------------
/// ------------------------------------------------------------

func parseEmailMessage(msg *f.EmailMessage) error {
	if msg.TemplateFS == nil {
		return fmt.Errorf("template_dir_required")
	}
	templateDir := msg.TemplateFS
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

		tpl, err = template.ParseFS(templateDir, fmt.Sprintf("templates/emails/%s.txt", msg.Template))
		if err != nil {
			return err
		}
		var textContent bytes.Buffer
		err = tpl.Execute(&textContent, msg.TemplateData)
		if err != nil {
			return err
		}
		msg.Text = textContent.String()
	}
	return nil
}

func NewEmailSender(provider string) f.EmailSender {
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
