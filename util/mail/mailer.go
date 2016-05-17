package mail

import (
	"fmt"
	"net/smtp"
)

type Mailer interface {
	DefaultMail() *Mail
	Server() chan<- *Mail
	SetSignature(string)
	Signature() string
	Send(*Mail) error
	SendAsync(*Mail) <-chan error
}

type SMTPMailer struct {
	host      string
	port      int
	auth      smtp.Auth
	sign      string
	makeplain bool
	*Mail     // contains the default settings
}

func (ml *SMTPMailer) SetHost(host string) {
	ml.host = host
}

func (ml *SMTPMailer) Host() string {
	if ml.host == "" {
		return "127.0.0.1"
	}
	return ml.host
}

func (ml *SMTPMailer) SetPort(port int) {
	ml.port = port
}

func (ml *SMTPMailer) Port() int {
	if ml.port == 0 {
		return 25
	}
	return ml.port
}

func (ml *SMTPMailer) SetAuth(auth smtp.Auth) {
	ml.auth = auth
}

func (ml *SMTPMailer) Auth() smtp.Auth {
	return ml.auth
}

func (ml *SMTPMailer) DefaultMail() *Mail {
	return ml.Mail
}

func (ml *SMTPMailer) Server() chan<- *Mail {
	// TODO: proper server that is able to send multiple mails using the same connection
	// and increase number of connections on heavy load
	c := make(chan *Mail, 50)

	go func() {
		for m := range c {
			go ml.Send(m)
		}
	}()

	return c
}

func (ml *SMTPMailer) SetSignature(sign string) {
	ml.sign = sign
}

func (ml *SMTPMailer) Signature() string {
	return ml.sign
}

/* TODO: makeplain setting to create text/plain message automatically for text/html only emails
func (ml *Mailer) GeneratePlainTextVersion(gen bool) {
	ml.makeplain = gen
}

func (ml *Mailer) PlainTextVersion() bool {
	return ml.makeplain
}
*/

func (ml *SMTPMailer) Send(m *Mail) error {
	if ml.Mail == nil {
		ml.Mail = &Mail{}
	}

	to := make([]string, 0, len(m.to)+len(m.cc)+len(m.bcc)+len(ml.to)+len(ml.cc)+len(ml.bcc))

	for _, addr := range m.to {
		to = append(to, addr.Address)
	}
	for _, addr := range m.cc {
		to = append(to, addr.Address)
	}
	for _, addr := range m.bcc {
		to = append(to, addr.Address)
	}
	for _, addr := range ml.to {
		to = append(to, addr.Address)
	}
	for _, addr := range ml.cc {
		to = append(to, addr.Address)
	}
	for _, addr := range ml.bcc {
		to = append(to, addr.Address)
	}

	from := m.BounceTo().Address

	if from == "" {
		from = ml.BounceTo().Address
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", ml.Host(), ml.Port()),
		ml.auth,
		from,
		to,
		buildMail(m, ml),
	)
}

func (ml *SMTPMailer) SendAsync(m *Mail) <-chan error {
	c := make(chan error, 1)

	go func() {
		c <- ml.Send(m)
	}()

	return c
}

func NewSMTPMailer(host string, port int, auth smtp.Auth) *SMTPMailer {
	return &SMTPMailer{
		Mail: &Mail{},
		host: host,
		port: port,
		auth: auth,
	}
}
