//+build go1.5

package mail

import (
	"bytes"
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

func (me *MessageEncoder) String() string {
	me.Writer.Close()
	if me.b.Len() > 0 {
		return me.b.String()
	}

	return ""
}

func (me *MessageEncoder) Bytes() []byte {
	me.Writer.Close()
	if me.b.Len() > 0 {
		return me.b.Bytes()
	}

	return []byte{}
}
