package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zaolab/sunnified/util/validate"
)

const DefaultAttachmentLimit = 25 * 1024 * 1024
const MaxInt = int64(^uint64(0) >> 1)

var ErrEmailInvalid = errors.New("invalid email address")
var ErrFileIsDir = errors.New("cannot attach directory")
var ErrFileInvalid = errors.New("invalid file")
var ErrAttachmentExceedLimit = errors.New("attachment is bigger than limit")

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

type Attachment struct {
	Name string
	Data []byte
}

type Addresses []mail.Address

func NewMail() *Mail {
	return &Mail{}
}

func (ad Addresses) String() string {
	b := make([]string, len(ad))
	for i, a := range ad {
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

func (m *Mail) SetHeader(key, value string) {
	if m.headers == nil {
		m.headers = make(textproto.MIMEHeader)
	}

	m.headers.Set(key, value)
}

func (m *Mail) AddHeader(key, value string) {
	if m.headers == nil {
		m.headers = make(textproto.MIMEHeader)
	}

	m.headers.Add(key, value)
}

func (m *Mail) GetHeader(key string) string {
	if m.headers == nil {
		m.headers = make(textproto.MIMEHeader)
		return ""
	}

	return m.headers.Get(key)
}

func (m *Mail) DelHeader(key string) {
	if m.headers == nil {
		m.headers = make(textproto.MIMEHeader)
		return
	}

	m.headers.Del(key)
}

func (m *Mail) AttachmentLimit() int64 {
	if m.attlimit == 0 {
		m.attlimit = DefaultAttachmentLimit
	} else if m.attlimit == -1 {
		m.attlimit = MaxInt
	}

	return m.attlimit
}

func (m *Mail) SetAttachmentLimit(l int64) {
	m.attlimit = l
}

func (m *Mail) SetTo(email string, name string) error {
	if validate.IsEmail(email) {
		m.to = Addresses{mail.Address{Name: name, Address: email}}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) AddTo(email, name string) error {
	if validate.IsEmail(email) {
		m.to = append(m.to, mail.Address{Name: name, Address: email})
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) To() Addresses {
	tmp := make(Addresses, len(m.to))
	copy(tmp, m.to)
	return tmp
}

func (m *Mail) SetFrom(email, name string) error {
	if validate.IsEmail(email) {
		m.from = mail.Address{Name: name, Address: email}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) From() mail.Address {
	return m.from
}

func (m *Mail) SetCc(email, name string) error {
	if validate.IsEmail(email) {
		m.cc = Addresses{mail.Address{Name: name, Address: email}}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) AddCc(email, name string) error {
	if validate.IsEmail(email) {
		m.cc = append(m.cc, mail.Address{Name: name, Address: email})
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) Cc() Addresses {
	tmp := make(Addresses, len(m.cc))
	copy(tmp, m.cc)
	return tmp
}

func (m *Mail) SetBcc(email, name string) error {
	if validate.IsEmail(email) {
		a := mail.Address{Name: name, Address: email}
		m.bcc = Addresses{a}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) AddBcc(email, name string) error {
	if validate.IsEmail(email) {
		a := mail.Address{Name: name, Address: email}
		m.bcc = append(m.bcc, a)
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) Bcc() Addresses {
	tmp := make(Addresses, len(m.bcc))
	copy(tmp, m.bcc)
	return tmp
}

func (m *Mail) SetReplyTo(email, name string) error {
	if validate.IsEmail(email) {
		m.replyto = mail.Address{Name: name, Address: email}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) ReplyTo() mail.Address {
	return m.replyto
}

func (m *Mail) SetBounceTo(email string) error {
	if validate.IsEmail(email) {
		m.bounce = mail.Address{Name: "", Address: email}
		return nil
	}

	return ErrEmailInvalid
}

func (m *Mail) BounceTo() mail.Address {
	if m.bounce.Address == "" {
		return m.from
	}

	return m.bounce
}

func (m *Mail) SetSubject(subj string) {
	// diff golang version output diff string if missiong email address
	// using a constant email address, it is then possible to determine the output
	s := mail.Address{Name: subj, Address: "a@abc.com"}
	subj = s.String()
	m.subject = strings.TrimSpace(strings.TrimSuffix(subj, "<a@abc.com>"))
	if lsub := len(m.subject); lsub > 0 && m.subject[0] == '"' && m.subject[lsub-1] == '"' {
		m.subject = strings.Replace(m.subject[1:lsub-1], `\"`, `"`, -1)
	}
}

func (m *Mail) Subject() string {
	return m.subject
}

func (m *Mail) SetMessage(mimetype, msg string) {
	if m.msg == nil {
		m.msg = make(map[string]string)
	}
	m.msg[mimetype] = msg
}

func (m *Mail) SetTextMessage(msg string) {
	m.SetMessage("text/plain", msg)
}

func (m *Mail) SetHTMLMessage(msg string) {
	m.SetMessage("text/html", msg)
}

func (m *Mail) AppendMessage(mimetype, msg string) {
	if m.msg == nil {
		m.msg = make(map[string]string)
	}

	if _, exists := m.msg[mimetype]; exists {
		m.msg[mimetype] += msg
	} else {
		m.msg[mimetype] = msg
	}
}

func (m *Mail) AppendTextMessage(msg string) {
	m.AppendMessage("text/plain", msg)
}

func (m *Mail) AppendHTMLMessage(msg string) {
	m.AppendMessage("text/html", msg)
}

func (m *Mail) Message() map[string]string {
	tmp := make(map[string]string)
	for k, v := range m.msg {
		tmp[k] = v
	}
	return tmp
}

func (m *Mail) AttachFile(fpath string) (err error) {
	var fd *os.File

	if fd, err = os.Open(fpath); err == nil {
		err = m.AttachFD(fd)
	}

	return
}

func (m *Mail) AttachFD(fd *os.File) error {
	if fd == nil {
		return ErrFileInvalid
	}

	var st os.FileInfo
	var err error

	if st, err = fd.Stat(); err == nil {
		if st.IsDir() {
			return ErrFileIsDir
		} else if st.Size()+m.attsize > m.AttachmentLimit() {
			return ErrAttachmentExceedLimit
		}
	} else {
		return err
	}

	fname := filepath.Base(fd.Name())

	if content, err := ioutil.ReadAll(fd); err == nil {
		b, size := EncodeBase64WithNewLine(content)
		m.attsize += size

		if m.attsize > m.AttachmentLimit() {
			m.attsize -= size
			return ErrAttachmentExceedLimit
		}

		m.att = append(m.att, Attachment{
			Name: fname,
			Data: b,
		})
	} else {
		return err
	}

	return nil
}

func (m *Mail) Attach(name string, b []byte) error {
	size := int64(len(b))

	if size+m.attsize > m.AttachmentLimit() {
		return ErrAttachmentExceedLimit
	}

	newb, _ := EncodeBase64WithNewLine(b)

	m.att = append(m.att, Attachment{
		Name: name,
		Data: newb,
	})

	m.attsize += size
	return nil
}

func (m *Mail) Attachments() []Attachment {
	att := make([]Attachment, len(m.att))
	copy(att, m.att)
	return m.att
}

func (m *Mail) String() string {
	return string(buildMail(m, nil))
}

func (m *Mail) Bytes() []byte {
	return buildMail(m, nil)
}

func (m *Mail) Clone() *Mail {
	headers := make(textproto.MIMEHeader)
	for k, v := range m.headers {
		value := make([]string, len(v))
		copy(value, v)
		headers[k] = value
	}

	return &Mail{
		to:       m.To(),
		from:     m.from,
		cc:       m.Cc(),
		bcc:      m.Bcc(),
		msg:      m.Message(),
		subject:  m.subject,
		att:      m.Attachments(),
		attlimit: m.attlimit,
		attsize:  m.attsize,
		headers:  headers,
		bounce:   m.bounce,
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
