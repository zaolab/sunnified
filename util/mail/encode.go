//+build !go1.5

package mail

const lenlimit = 76
const upperhex = "0123456789ABCDEF"

var softNl = []byte{'=', '\r', '\n'}

func EncodeQuotedPrintableString(value string) string {
	enc := MessageEncoder{}
	enc.Write([]byte(value))
	return enc.String()
}

func EncodeQuotedPrintable(value []byte) []byte {
	enc := MessageEncoder{}
	enc.Write(value)
	return enc.Bytes()
}

func NewMessageEncoder() *MessageEncoder {
	return &MessageEncoder{}
}

type MessageEncoder struct {
	b []byte
	i int
}

func (me *MessageEncoder) Write(p []byte) (int, error) {
	var (
		b    byte
		iscr bool
		issp bool
		lb   int
		lp   int
	)

	if lb, lp = len(me.b), len(p); me.b == nil || cap(me.b)-lb < lp {
		newb := make([]byte, lb, lb+lp*3)
		copy(newb, me.b)
		me.b = newb
	}

	for _, b = range p {
		switch {
		case b == '\r':
			iscr = true
			continue
		case b == '\n' || iscr:
			me.insertNl(issp)
			iscr = false
			issp = false
		case b == ' ' || b == '\t' || (b >= '!' && b <= '~' && b != '='):
			if me.i+1 >= lenlimit {
				me.b = append(me.b, softNl...)
				me.i = 0
			}
			me.b = append(me.b, b)
			me.i += 1
			issp = b == ' ' || b == '\t'
		default:
			if me.i+3 >= lenlimit {
				me.b = append(me.b, softNl...)
				me.i = 0
			}
			me.b = append(me.b, encode(b)...)
			me.i += 3
			issp = false
		}
	}

	if iscr {
		me.insertNl(issp)
	}

	return lp, nil
}

func (me *MessageEncoder) insertNl(issp bool) {
	if issp {
		lastl := len(me.b) - 1
		chr := me.b[lastl]
		sp := encode(chr)

		if me.i+2 > lenlimit {
			me.b[lastl] = softNl[0]
			me.b = append(me.b, softNl[1], softNl[2], sp[0], sp[1], sp[2])
		} else {
			me.b[lastl] = sp[0]
			me.b = append(me.b, sp[1], sp[2])
		}
	}

	me.b = append(me.b, '\r', '\n')
	me.i = 0
}

func (me *MessageEncoder) String() (s string) {
	me.i = 0
	s = string(me.b)
	me.b = nil
	return
}

func (me *MessageEncoder) Bytes() (b []byte) {
	me.i = 0
	b = me.b
	me.b = nil
	return
}

func encode(b byte) []byte {
	return []byte{'=', upperhex[b>>4], upperhex[b&0x0f]}
}
