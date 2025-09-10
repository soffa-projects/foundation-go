package f

import (
	"html/template"
	"io/fs"
)

type EmailSender interface {
	Send(msg EmailMessage) error
	On(name string, fn func(msg EmailMessage))
}

type Attachment struct {
	Filename    string
	Data        []byte
	ContentType string
}

type EmailMessage struct {
	TemplateFS   fs.FS
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
