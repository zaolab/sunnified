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

func (this *SMTPMailer) SetHost(host string) {
	this.host = host
}

func (this *SMTPMailer) Host() string {
	if this.host == "" {
		return "127.0.0.1"
	}
	return this.host
}

func (this *SMTPMailer) SetPort(port int) {
	this.port = port
}

func (this *SMTPMailer) Port() int {
	if this.port == 0 {
		return 25
	}
	return this.port
}

func (this *SMTPMailer) SetAuth(auth smtp.Auth) {
	this.auth = auth
}

func (this *SMTPMailer) Auth() smtp.Auth {
	return this.auth
}

func (this *SMTPMailer) DefaultMail() *Mail {
	return this.Mail
}

func (this *SMTPMailer) Server() chan<- *Mail {
	// TODO: proper server that is able to send multiple mails using the same connection
	// and increase number of connections on heavy load
	c := make(chan *Mail, 50)

	go func() {
		for m := range c {
			go this.Send(m)
		}
	}()

	return c
}

func (this *SMTPMailer) SetSignature(sign string) {
	this.sign = sign
}

func (this *SMTPMailer) Signature() string {
	return this.sign
}

/* TODO: makeplain setting to create text/plain message automatically for text/html only emails
func (this *Mailer) GeneratePlainTextVersion(gen bool) {
	this.makeplain = gen
}

func (this *Mailer) PlainTextVersion() bool {
	return this.makeplain
}
*/

func (this *SMTPMailer) Send(m *Mail) error {
	if this.Mail == nil {
		this.Mail = &Mail{}
	}

	to := make([]string, 0, len(m.to)+len(m.cc)+len(m.bcc)+len(this.to)+len(this.cc)+len(this.bcc))

	for _, addr := range m.to {
		to = append(to, addr.Address)
	}
	for _, addr := range m.cc {
		to = append(to, addr.Address)
	}
	for _, addr := range m.bcc {
		to = append(to, addr.Address)
	}
	for _, addr := range this.to {
		to = append(to, addr.Address)
	}
	for _, addr := range this.cc {
		to = append(to, addr.Address)
	}
	for _, addr := range this.bcc {
		to = append(to, addr.Address)
	}

	from := m.BounceTo().Address

	if from == "" {
		from = this.BounceTo().Address
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", this.Host(), this.Port()),
		this.auth,
		from,
		to,
		buildMail(m, this),
	)
}

func (this *SMTPMailer) SendAsync(m *Mail) <-chan error {
	c := make(chan error, 1)

	go func() {
		c <- this.Send(m)
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
