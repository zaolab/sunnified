//+build go1.5

package mail

import (
	"bytes"
	"io/ioutil"
	"mime/quotedprintable"
)

func EncodeQuotedPrintableString(value string) string {
	b := bytes.Buffer{}
	enc := MessageEncoder{
		&b,
		quotedprintable.NewWriter(&b),
	}
	enc.Write([]byte(value))
	return enc.String()
}

func EncodeQuotedPrintable(value []byte) []byte {
	b := bytes.Buffer{}
	enc := MessageEncoder{
		&b,
		quotedprintable.NewWriter(&b),
	}
	enc.Write(value)
	return enc.Bytes()
}

func NewMessageEncoder() *MessageEncoder {
	b := &bytes.Buffer{}
	return &MessageEncoder{
		b,
		quotedprintable.NewWriter(b),
	}
}

type MessageEncoder struct {
	b *bytes.Buffer
	*quotedprintable.Writer
}

func (this *MessageEncoder) String() string {
	this.Writer.Close()
	if this.b.Len() > 0 {
		return this.b.String()
	} else {
		return ""
	}
}

func (this *MessageEncoder) Bytes() []byte {
	this.Writer.Close()
	if this.b.Len() > 0 {
		return this.b.Bytes()
	} else {
		return ""
	}
}
