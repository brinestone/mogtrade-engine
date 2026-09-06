package mail

import (
	"context"
	"encoding/base64"
	"io"

	"github.com/mailtrap/mailtrap-go"
)

type mailtrapMailer struct {
	sender string
	name   string
	client *mailtrap.Client
}

type mailtrapMailSendResult mailtrap.SendResponse

func (m mailtrapMailSendResult) Successful() bool {
	return m.Success
}

func (m mailtrapMailSendResult) GetMailId() string {
	return m.MessageIDs[0]
}

func toMailTrapRecepientAddresses(mail MailDescription) []mailtrap.Address {
	res := make([]mailtrap.Address, 0)
	for _, m := range mail.Recepients {
		res = append(res, mailtrap.Address{Email: m})
	}
	return res
}

func toMailTrapAttachment(a AttachmentDescription) (mailtrap.Attachment, error) {
	buf, err := io.ReadAll(a.Reader)
	if err != nil {
		return mailtrap.Attachment{}, err
	}

	data := base64.StdEncoding.EncodeToString(buf)
	return mailtrap.Attachment{
		Type:        a.Type,
		Content:     data,
		Filename:    a.Name,
		Disposition: "attachment",
	}, nil
}

func toMailTrapRequest(mail MailDescription, name, sender string) (*mailtrap.SendRequest, error) {
	req := &mailtrap.SendRequest{
		From:    mailtrap.Address{Name: name, Email: sender},
		To:      toMailTrapRecepientAddresses(mail),
		Subject: mail.Subject,
	}
	if mail.IsHtml {
		req.HTML = string(mail.Content)
	} else {
		req.Text = string(mail.Content)
	}

	for _, a := range mail.Attachments {
		att, err := toMailTrapAttachment(a)
		if err != nil {
			return nil, err
		}
		req.Attachments = append(req.Attachments, att)
	}
	return req, nil
}

// SendBulk implements [Mailer].
func (m *mailtrapMailer) SendMail(ctx context.Context, mail MailDescription) (MailDeliveryResult, error) {
	req, err := toMailTrapRequest(mail, m.name, m.sender)
	if err != nil {
		return nil, err
	}
	res, _, err := m.client.Send(ctx, req)
	if err != nil {
		return nil, err
	}

	return mailtrapMailSendResult(*res), nil
}

// SendMail implements [Mailer].
func (m *mailtrapMailer) SendBulk(context.Context, []MailDescription) (MailDeliveryResult, error) {
	panic("unimplemented")
}

func NewMailtrapMailer(sender string, displayName string, keyProvider func() string) (Mailer, error) {
	client, err := mailtrap.NewClient(keyProvider())
	if err != nil {
		return nil, err
	}

	return &mailtrapMailer{
		sender: sender,
		name:   displayName,
		client: client,
	}, nil
}
