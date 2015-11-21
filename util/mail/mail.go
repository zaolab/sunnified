package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/zaolab/sunnified/util/validate"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DEFAULT_ATTACHMENT_LIMIT = 25 * 1024 * 1024
const MAX_INT = int64(^uint64(0) >> 1)

var ErrEmailInvalid = errors.New("Invalid email address")
var ErrFileIsDir = errors.New("Cannot attach directory")
var ErrFileInvalid = errors.New("Invalid file")
var ErrAttachmentExceedLimit = errors.New("Attachment is bigger than limit")

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

type Attachment struct {
	Name string
	Data []byte
}

type Addresses []mail.Address

func NewMail() *Mail {
	return &Mail{}
}

func (this Addresses) String() string {
	b := make([]string, len(this))
	for i, a := range this {
		b[i] = a.String()
	}
	return strings.Join(b, ", ")
}

// TODO: add inline attachment (mainly for inline images) support
type Mail struct {
	to       Addresses
	from     mail.Address
	replyto  mail.Address
	cc       Addresses
	bcc      Addresses
	bounce   mail.Address
	headers  textproto.MIMEHeader
	msg      map[string]string
	subject  string
	att      []Attachment
	attlimit int64
	attsize  int64
}

func (this *Mail) SetHeader(key, value string) {
	if this.headers == nil {
		this.headers = make(textproto.MIMEHeader)
	}

	this.headers.Set(key, value)
}

func (this *Mail) AddHeader(key, value string) {
	if this.headers == nil {
		this.headers = make(textproto.MIMEHeader)
	}

	this.headers.Add(key, value)
}

func (this *Mail) GetHeader(key string) string {
	if this.headers == nil {
		this.headers = make(textproto.MIMEHeader)
		return ""
	}

	return this.headers.Get(key)
}

func (this *Mail) DelHeader(key string) {
	if this.headers == nil {
		this.headers = make(textproto.MIMEHeader)
		return
	}

	this.headers.Del(key)
}

func (this *Mail) AttachmentLimit() int64 {
	if this.attlimit == 0 {
		this.attlimit = DEFAULT_ATTACHMENT_LIMIT
	} else if this.attlimit == -1 {
		this.attlimit = MAX_INT
	}

	return this.attlimit
}

func (this *Mail) SetAttachmentLimit(l int64) {
	this.attlimit = l
}

func (this *Mail) SetTo(email string, name string) error {
	if validate.IsEmail(email) {
		this.to = Addresses{mail.Address{Name: name, Address: email}}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) AddTo(email, name string) error {
	if validate.IsEmail(email) {
		this.to = append(this.to, mail.Address{Name: name, Address: email})
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) To() Addresses {
	tmp := make(Addresses, len(this.to))
	copy(tmp, this.to)
	return tmp
}

func (this *Mail) SetFrom(email, name string) error {
	if validate.IsEmail(email) {
		this.from = mail.Address{Name: name, Address: email}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) From() mail.Address {
	return this.from
}

func (this *Mail) SetCc(email, name string) error {
	if validate.IsEmail(email) {
		this.cc = Addresses{mail.Address{Name: name, Address: email}}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) AddCc(email, name string) error {
	if validate.IsEmail(email) {
		this.cc = append(this.cc, mail.Address{Name: name, Address: email})
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) Cc() Addresses {
	tmp := make(Addresses, len(this.cc))
	copy(tmp, this.cc)
	return tmp
}

func (this *Mail) SetBcc(email, name string) error {
	if validate.IsEmail(email) {
		a := mail.Address{Name: name, Address: email}
		this.bcc = Addresses{a}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) AddBcc(email, name string) error {
	if validate.IsEmail(email) {
		a := mail.Address{Name: name, Address: email}
		this.bcc = append(this.bcc, a)
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) Bcc() Addresses {
	tmp := make(Addresses, len(this.bcc))
	copy(tmp, this.bcc)
	return tmp
}

func (this *Mail) SetReplyTo(email, name string) error {
	if validate.IsEmail(email) {
		this.replyto = mail.Address{Name: name, Address: email}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) ReplyTo() mail.Address {
	return this.replyto
}

func (this *Mail) SetBounceTo(email string) error {
	if validate.IsEmail(email) {
		this.bounce = mail.Address{Name: "", Address: email}
		return nil
	} else {
		return ErrEmailInvalid
	}
}

func (this *Mail) BounceTo() mail.Address {
	if this.bounce.Address == "" {
		return this.from
	} else {
		return this.bounce
	}
}

func (this *Mail) SetSubject(subj string) {
	// diff golang version output diff string if missiong email address
	// using a constant email address, it is then possible to determine the output
	s := mail.Address{Name: subj, Address: "a@abc.com"}
	subj = s.String()
	this.subject = strings.TrimSpace(strings.TrimSuffix(subj, "<a@abc.com>"))
}

func (this *Mail) Subject() string {
	return this.subject
}

func (this *Mail) SetMessage(mimetype, msg string) {
	if this.msg == nil {
		this.msg = make(map[string]string)
	}
	this.msg[mimetype] = msg
}

func (this *Mail) SetTextMessage(msg string) {
	this.SetMessage("text/plain", msg)
}

func (this *Mail) SetHTMLMessage(msg string) {
	this.SetMessage("text/html", msg)
}

func (this *Mail) AppendMessage(mimetype, msg string) {
	if this.msg == nil {
		this.msg = make(map[string]string)
	}

	if _, exists := this.msg[mimetype]; exists {
		this.msg[mimetype] += msg
	} else {
		this.msg[mimetype] = msg
	}
}

func (this *Mail) AppendTextMessage(msg string) {
	this.AppendMessage("text/plain", msg)
}

func (this *Mail) AppendHTMLMessage(msg string) {
	this.AppendMessage("text/html", msg)
}

func (this *Mail) Message() map[string]string {
	tmp := make(map[string]string)
	for k, v := range this.msg {
		tmp[k] = v
	}
	return tmp
}

func (this *Mail) AttachFile(fpath string) (err error) {
	var fd *os.File

	if fd, err = os.Open(fpath); err == nil {
		err = this.AttachFD(fd)
	}

	return
}

func (this *Mail) AttachFD(fd *os.File) error {
	if fd == nil {
		return ErrFileInvalid
	}

	var st os.FileInfo
	var err error

	if st, err = fd.Stat(); err == nil {
		if st.IsDir() {
			return ErrFileIsDir
		} else if st.Size()+this.attsize > this.AttachmentLimit() {
			return ErrAttachmentExceedLimit
		}
	} else {
		return err
	}

	fname := filepath.Base(fd.Name())

	if content, err := ioutil.ReadAll(fd); err == nil {
		b, size := EncodeBase64WithNewLine(content)
		this.attsize += size

		if this.attsize > this.AttachmentLimit() {
			this.attsize -= size
			return ErrAttachmentExceedLimit
		}

		this.att = append(this.att, Attachment{
			Name: fname,
			Data: b,
		})
	} else {
		return err
	}

	return nil
}

func (this *Mail) Attach(name string, b []byte) error {
	size := int64(len(b))

	if size+this.attsize > this.AttachmentLimit() {
		return ErrAttachmentExceedLimit
	}

	newb, _ := EncodeBase64WithNewLine(b)

	this.att = append(this.att, Attachment{
		Name: name,
		Data: newb,
	})

	this.attsize += size
	return nil
}

func (this *Mail) Attachments() []Attachment {
	att := make([]Attachment, len(this.att))
	copy(att, this.att)
	return this.att
}

func (this *Mail) String() string {
	return string(buildMail(this, nil))
}

func (this *Mail) Bytes() []byte {
	return buildMail(this, nil)
}

func (this *Mail) Clone() *Mail {
	headers := make(textproto.MIMEHeader)
	for k, v := range this.headers {
		value := make([]string, len(v))
		copy(value, v)
		headers[k] = value
	}

	return &Mail{
		to:       this.To(),
		from:     this.from,
		cc:       this.Cc(),
		bcc:      this.Bcc(),
		msg:      this.Message(),
		subject:  this.subject,
		att:      this.Attachments(),
		attlimit: this.attlimit,
		attsize:  this.attsize,
		headers:  headers,
		bounce:   this.bounce,
	}
}

func buildMail(mail *Mail, mailer Mailer) []byte {
	var (
		defmail   Mail
		signature string
		matt      []Attachment
	)

	if mailer != nil {
		defmail = *mailer.DefaultMail()
		signature = mailer.Signature()
		matt = defmail.att
	}

	var (
		mbuf        = bytes.Buffer{}
		boundary    [24]byte
		boundaryID  int
		boundaryStr string
		from        = mail.from
		to          = append(mail.To(), defmail.to...)
		cc          = append(mail.Cc(), defmail.cc...)
		replyto     = mail.replyto
		subject     = mail.subject
	)

	if from.Address == "" {
		from = defmail.from
	}
	if replyto.Address == "" {
		replyto = defmail.replyto
	}
	if subject == "" {
		subject = defmail.subject
	}

	rand.Read(boundary[:])
	boundaryStr = fmt.Sprintf("_%x_%%d_", boundary)
	fmt.Fprintf(&mbuf, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\n",
		from.String(),
		to.String(),
		subject,
		time.Now().Format(time.RFC1123Z))

	if len(cc) > 0 {
		fmt.Fprintf(&mbuf, "Cc: %s\r\n", cc.String())
	}
	if replyto.Address != "" {
		fmt.Fprintf(&mbuf, "Reply-To: %s\r\n", replyto.String())
	}

	mail.headers.Del("Content-Type")
	mail.headers.Del("Content-Transfer-Encoding")

	for k, v := range mail.headers {
		for _, value := range v {
			fmt.Fprintf(&mbuf, "%s: %s\r\n", k, value)
		}
	}

	bound := fmt.Sprintf(boundaryStr, boundaryID)

	if len(mail.att) > 0 || len(matt) > 0 {
		fmt.Fprintf(&mbuf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", bound)
		boundaryID++

		mwriter := multipart.NewWriter(&mbuf)
		mwriter.SetBoundary(bound)

		if header, message := buildMessage(mail.msg, signature, fmt.Sprintf(boundaryStr, boundaryID)); message != nil {
			if p, err := mwriter.CreatePart(header); err == nil {
				p.Write(message)
			}
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Transfer-Encoding", "base64")

		appendAtt := func(att []Attachment) {
			for _, att := range att {
				h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`,
					quoteEscaper.Replace(att.Name)))

				t := mime.TypeByExtension(filepath.Ext(att.Name))
				if t == "" {
					t = "application/octet-stream"
				}
				h.Set("Content-Type", t)

				if p, err := mwriter.CreatePart(h); err == nil {
					p.Write(att.Data)
				}
			}
		}

		appendAtt(mail.att)
		appendAtt(matt)

		mwriter.Close()
	} else if header, message := buildMessage(mail.msg, signature, bound); message != nil {
		for k, v := range header {
			for _, value := range v {
				fmt.Fprintf(&mbuf, "%s: %s\r\n", k, value)
			}
		}

		mbuf.WriteString("\r\n")
		mbuf.Write(message)
	}

	return mbuf.Bytes()
}

func buildMessage(msg map[string]string, sign, bound string) (textproto.MIMEHeader, []byte) {
	var (
		lmsg    = len(msg)
		h       = make(textproto.MIMEHeader)
		mbuf    = bytes.Buffer{}
		charset = `; charset="utf-8"`
	)

	if lmsg == 0 {
		return nil, nil
	}

	if lmsg > 1 {
		mwriter := multipart.NewWriter(&mbuf)
		mwriter.SetBoundary(bound)

		h.Set("Content-Transfer-Encoding", "quoted-printable")

		for t, v := range msg {
			if !strings.Contains(t, "charset") {
				t = t + charset
			}

			h.Set("Content-Type", t)

			if p, err := mwriter.CreatePart(h); err == nil {
				p.Write(EncodeQuotedPrintable(MergeMessageWithSignature(v, t, sign)))
			}
		}

		mwriter.Close()

		h.Del("Content-Transfer-Encoding")
		h.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, bound))
	} else if lmsg == 1 {
		var t, v string
		for t, v = range msg {
		}

		if !strings.Contains(t, "charset") {
			t = t + charset
		}

		h.Set("Content-Transfer-Encoding", "quoted-printable")
		h.Set("Content-Type", t)
		mbuf.Write(EncodeQuotedPrintable(MergeMessageWithSignature(v, t, sign)))
	}

	return h, mbuf.Bytes()
}

var nl2Br = strings.NewReplacer("\r\n", "<br>", "\r", "<br>", "\n", "<br>")

func MergeMessageWithSignature(msg, mimetype, sign string) []byte {
	var newb []byte
	if sign != "" {
		if strings.HasPrefix(mimetype, "text/html") {
			sign = nl2Br.Replace(sign)
			sign = "<br><br>" + sign
		} else {
			sign = "\r\n\r\n" + sign
		}

		lv := len(msg)
		lall := lv + len(sign)

		newb = make([]byte, lall)
		copy(newb, msg)
		copy(newb[lv:lall], sign)
	} else {
		newb = []byte(msg)
	}
	return newb
}

func EncodeBase64WithNewLine(content []byte) (b []byte, size int64) {
	var lcontent int

	if lcontent = len(content); lcontent != 0 {
		lines := (lcontent + 56) / 57
		linesM1 := lines - 1
		lb := base64.StdEncoding.EncodedLen(lcontent) + (linesM1 * 2)
		b = make([]byte, lb)
		s := 0
		e := 76
		s2 := 0
		e2 := 57

		for i := 0; i < linesM1; i++ {
			base64.StdEncoding.Encode(b[s:e], content[s2:e2])
			b[e] = '\r'
			b[e+1] = '\n'

			s = e + 2
			e += 78
			s2 = e2
			e2 += 57
		}

		base64.StdEncoding.Encode(b[s:lb], content[s2:lcontent])
	}

	return b, int64(lcontent)
}
