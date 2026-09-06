package mail

import (
	"context"
	"io"
)

type AttachmentDescription struct {
	Name   string
	Type   string
	Reader io.Reader
}

type MailDescription struct {
	Recepients  []string
	Subject     string
	IsHtml      bool
	Content     []byte
	Attachments []AttachmentDescription
}

type MailDeliveryResult interface {
	GetMailId() string
	Successful() bool
}

type Mailer interface {
	SendMail(context.Context, MailDescription) (MailDeliveryResult, error)
	SendBulk(context.Context, []MailDescription) (MailDeliveryResult, error)
}
