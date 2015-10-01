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

func (this *MessageEncoder) Write(p []byte) (int, error) {
	var (
		b    byte
		iscr bool
		issp bool
		lb   int
		lp   int
	)

	if lb, lp = len(this.b), len(p); this.b == nil || cap(this.b)-lb < lp {
		newb := make([]byte, lb, lb+lp*3)
		copy(newb, this.b)
		this.b = newb
	}

	for _, b = range p {
		switch {
		case b == '\r':
			iscr = true
			continue
		case b == '\n' || iscr:
			this.insertNl(issp)
			iscr = false
			issp = false
		case b == ' ' || b == '\t' || (b >= '!' && b <= '~' && b != '='):
			if this.i+1 >= lenlimit {
				this.b = append(this.b, softNl...)
				this.i = 0
			}
			this.b = append(this.b, b)
			this.i += 1
			issp = b == ' ' || b == '\t'
		default:
			if this.i+3 >= lenlimit {
				this.b = append(this.b, softNl...)
				this.i = 0
			}
			this.b = append(this.b, encode(b)...)
			this.i += 3
			issp = false
		}
	}

	if iscr {
		this.insertNl(issp)
	}

	return lp, nil
}

func (this *MessageEncoder) insertNl(issp bool) {
	if issp {
		lastl := len(this.b) - 1
		chr := this.b[lastl]
		sp := encode(chr)

		if this.i+2 > lenlimit {
			this.b[lastl] = softNl[0]
			this.b = append(this.b, softNl[1], softNl[2], sp[0], sp[1], sp[2])
		} else {
			this.b[lastl] = sp[0]
			this.b = append(this.b, sp[1], sp[2])
		}
	}

	this.b = append(this.b, '\r', '\n')
	this.i = 0
}

func (this *MessageEncoder) String() (s string) {
	this.i = 0
	s = string(this.b)
	this.b = nil
	return
}

func (this *MessageEncoder) Bytes() (b []byte) {
	this.i = 0
	b = this.b
	this.b = nil
	return
}

func encode(b byte) []byte {
	return []byte{'=', upperhex[b>>4], upperhex[b&0x0f]}
}
