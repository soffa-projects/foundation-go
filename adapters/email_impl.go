package adapters

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/resend/resend-go/v2"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
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
		tpl, err := template.ParseFS(templateDir, fmt.Sprintf("emails/%s.html", msg.Template))
		if err != nil {
			return err
		}
		var htmlContent bytes.Buffer
		err = tpl.Execute(&htmlContent, msg.TemplateData)
		if err != nil {
			return err
		}
		msg.Html = htmlContent.String()

		tpl, err = template.ParseFS(templateDir, fmt.Sprintf("emails/%s.txt", msg.Template))
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

func NewEmailSender(senderName string, provider string) f.EmailSender {
	cfg, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse email provider: %v", err)
	}
	sender := cfg.Query("from")
	if sender == "" {
		sender := cfg.Query("sender")
		if sender == "" {
			log.Fatal("sender is required")
		}
	}

	fullSender := fmt.Sprintf("%s <%s>", senderName, sender)
	switch cfg.Scheme {
	case "resend":
		return newResendEmailProvider(cfg.User, fullSender)
	case "faker":
		return newFakeEmailProvider("", fullSender)
	}
	log.Fatal("unsupported email provider: %s", cfg.Scheme)
	return nil
}
