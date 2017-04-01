package sec

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

const (
	//dSecurityKey		    = "\x69\x51\xe8\x41\x50\x83\x19\xa4\xf0\x2f\xac\x7d\x99\xb7\x5e\xbe\x7e\x32\xf5\xa5\xf7\x1f\x43\x04\x96\xdd\x1b\xf0\x93\x4e\xc5\x44"
	//dCSRFToken			= "\xc7\x58\xa7\xf2\x15\x79\x54\x34\x24\xeb\x45\x50\x33\x0f\xa5\x52\x95\x36\x06\xb0\xb7\xdb\x5d\xa7\x07\xcf\xa5\x1c\x10\xe7\x4b\xd4"
	//dHashSalt			= "\x5d\xfb\xcf\x47\x30\xce\x2e\x43\xfa\x1c\x5f\xee\x76\x0f\xd7\x31\x14\x07\x24\xa8\xbf\xd0\x3c\x88\xfc\xa3\xdc\x3b\xae\xaa\x3a\x15"
	//dCSRFTokenLife		= 14400
	CSRFTokenMinLife       = 3600
	CSRFDefaultTokenLife   = 14400
	CSRFDefaultCookieName  = "XSRF-TOKEN"
	CSRFDefaultRequestName = "X-XSRF-TOKEN"
	CSRFTimestampLen       = 5
	CSRFRandTokenLen       = 16
)

// the resultant hash length should be longer than or equals to sessEntropy
var sessHash = sha256.New

// use a higher entropy (bytes) to prevent brute force session attack
var sessEnthropy = 24

type CSRFGate struct {
	config CSRFGateConfig
}

type CSRFGateConfig struct {
	SunnyConfig bool `config.namespace:"sunnified.sec.csrf"`
	Key         []byte
	Token       []byte
	Tokenlife   int    `config.default:"14400"`
	Cookiename  string `config.default:"XSRF-TOKEN"`
	Reqname     string `config.default:"X-XSRF-TOKEN"`
}

func NewCSRFGate(settings CSRFGateConfig) *CSRFGate {
	if settings.Key == nil || settings.Token == nil {
		return nil
	}

	if settings.Tokenlife == 0 {
		settings.Tokenlife = CSRFDefaultTokenLife
	}
	if settings.Reqname == "" {
		settings.Reqname = CSRFDefaultRequestName
	}
	if settings.Cookiename == "" {
		settings.Cookiename = CSRFDefaultCookieName
	}

	return &CSRFGate{config: settings}
}

type CSRFRequestBody struct {
	Name   string
	Value  string
	Cookie *http.Cookie
	Ok     bool
}

// SetCSRFToken returns a CsrfRequestBody containing the name and value to be used
// as a query string or form input that can be verified by VerifyCSRFToken.
// Additionally, a cookie will be set (if ResponseWriter is not nil) to cross authenticate validity of token data if non exists
func (cg *CSRFGate) CSRFToken(w http.ResponseWriter, r *http.Request) (crb CSRFRequestBody) {
	var (
		randToken   []byte
		msg         []byte
		writeCookie = false
		tstamp      = time.Now().Unix()
		// the current rolling global token.
		// this token is the share for the entire application
		// it rolls over to a new token every "csrf-token-life"
		currentToken = cg.csrfCurrentToken(tstamp)
		ckie, err    = r.Cookie(cg.config.Cookiename)
	)

	// gets the cookie containing the random token generated
	// the random token will be shared for all requests from the same machine/browser
	// this is a very simple mechanism for unique user identification
	if err == nil {
		randToken, err = AesCtrDecryptBase64(cg.config.Key, ckie.Value)
	}
	// if there are no random token from the cookie,
	// generate a new one ourselves.
	if err != nil || len(randToken) != CSRFRandTokenLen {
		randToken = GenRandomBytes(CSRFRandTokenLen)

		if randToken == nil {
			// the randomness of this token is not as critical to security
			lenToFill := CSRFRandTokenLen
			msgToHash := []byte(strconv.FormatInt(tstamp, 10))
			msgToHash = append(msgToHash, currentToken...)
			randToken = make([]byte, 0, CSRFRandTokenLen)

			// fill the random token slice using sha512 checksum
			// if random token exceeds 64 bytes(len of sha512),
			// it loops and generate more checksum to fill
			for lenToFill > 0 {
				h := sha512.Sum512(msgToHash)

				fillLen := lenToFill
				if fillLen > 64 {
					fillLen = 64
				}
				lenToFill = lenToFill - fillLen

				randToken = append(randToken, h[0:fillLen]...)
				msgToHash = h[:]
			}
		}

		// a new random token is generated, set the cookie to update it
		writeCookie = true
	}

	msg = make([]byte, 0, CSRFTimestampLen+CSRFRandTokenLen+len(currentToken))
	buf := bytes.NewBuffer(msg)
	binary.Write(buf, binary.LittleEndian, tstamp)

	// the csrf token consists of timestamp(5 bytes),
	// random bytes(16 bytes),
	// rolling global token(20bytes [sha1 checksum])
	msg = append(msg[0:CSRFTimestampLen], randToken...)
	msg = append(msg, currentToken...)

	if value, err := AesCtrEncryptBase64(cg.config.Key, msg); err == nil {
		if writeCookie {
			enc, err := AesCtrEncryptBase64(cg.config.Key, randToken)

			if err != nil {
				return
			}

			ckie = &http.Cookie{
				Name:  cg.config.Cookiename,
				Value: enc,
				Path:  "/",
			}
			if w != nil {
				http.SetCookie(w, ckie)
			}
		}

		crb.Name = cg.config.Reqname
		crb.Value = value
		crb.Cookie = ckie
		crb.Ok = true
	}

	return
}

// VerifyCSRFToken checks whether the request r includes a valid CSRF token
func (cg *CSRFGate) VerifyCSRFToken(r *http.Request) (valid bool) {
	var token string

	if token = r.Header.Get(cg.config.Reqname); token != "" {
		// TODO: for cross domain, the request will first perform an OPTIONS
		// with Access-Control-Request-Headers: X-XSRF-TOKEN
		// we gotten respond with Access-Control-Allow-Headers: X-XSRF-TOKEN somehow
		// if router doesn't respond by mirroring the request
		if ckie, err := r.Cookie(cg.config.Cookiename); err == nil {
			valid = token == ckie.Value
		}
	} else {
		r.ParseForm()
		token = r.Form.Get(cg.config.Reqname)

		if token == "" {
			return
		}

		result, err := AesCtrDecryptBase64(cg.config.Key, token)

		if err != nil || len(result) <= (CSRFTimestampLen+CSRFRandTokenLen) {
			return
		}

		lenTNC := CSRFTimestampLen + CSRFRandTokenLen

		tcreatedcap := CSRFTimestampLen
		if tcreatedcap < 8 {
			tcreatedcap = 8
		}

		tcreated := make([]byte, CSRFTimestampLen, tcreatedcap)

		// copy into a new slice, append overwrites original slice data
		copy(tcreated, result[0:CSRFTimestampLen])
		ckietoken := make([]byte, CSRFRandTokenLen)
		copy(ckietoken, result[CSRFTimestampLen:lenTNC])
		reqtoken := make([]byte, len(result)-lenTNC)
		copy(reqtoken, result[lenTNC:])

		if CSRFTimestampLen < 8 {
			filler := make([]byte, 8-CSRFTimestampLen)
			tcreated = append(tcreated, filler...)
		}

		var tcreated64 int64
		binary.Read(bytes.NewBuffer(tcreated), binary.LittleEndian, &tcreated64)
		tstamp := time.Now().Unix()

		// check whether request token has already expired
		if (tcreated64+int64(cg.config.Tokenlife)) < tstamp || tcreated64 > tstamp {
			return
		}

		// cookie authentication of csrf token is needed to ensure each machine has unique token
		if ckie, err := r.Cookie(cg.config.Cookiename); err == nil {
			dec, err := AesCtrDecryptBase64(cg.config.Key, ckie.Value)

			if err != nil || !bytes.Equal(dec, ckietoken) {
				return
			}
		} else {
			return
		}

		valid = bytes.Equal(reqtoken, cg.csrfCurrentToken(tstamp)) || bytes.Equal(reqtoken, cg.csrfPrevToken(tstamp))
	}

	return
}

func (cg *CSRFGate) csrfCurrentToken(t ...int64) []byte {
	var tnow int64

	if len(t) > 0 {
		tnow = t[0]
	} else {
		tnow = time.Now().Unix()
	}
	iteration := tnow / int64(cg.config.Tokenlife)
	return cg.csrfIterToken(iteration)
}

func (cg *CSRFGate) csrfCurrentTokenString(t ...int64) string {
	return string(cg.csrfCurrentToken(t...))
}

func (cg *CSRFGate) csrfPrevToken(t ...int64) []byte {
	var tnow int64

	if len(t) > 0 {
		tnow = t[0]
	} else {
		tnow = time.Now().Unix()
	}

	iteration := tnow / int64(cg.config.Tokenlife)
	return cg.csrfIterToken(iteration - 1)
}

func (cg *CSRFGate) csrfPrevTokenString(t ...int64) string {
	return string(cg.csrfPrevToken(t...))
}

func (cg *CSRFGate) csrfIterToken(iteration int64) []byte {
	itertoken := strconv.FormatInt(iteration, 10)
	h := hmac.New(sha1.New, cg.config.Token)
	h.Write([]byte(itertoken))
	hash := make([]byte, 0, h.Size())
	hash = h.Sum(hash)
	return hash
}

func (cg *CSRFGate) csrfIterTokenString(iteration int64) string {
	return string(cg.csrfIterToken(iteration))
}

// GenRandomBytes return a slice of random bytes of length l
func GenRandomBytes(l int) (rb []byte) {
	rb = make([]byte, l)

	// rand.Read() is blocking
	if _, err := rand.Read(rb); err != nil {
		rb = nil
	}

	return
}

func GenRandomString(l int) string {
	rb := GenRandomBytes(l)

	if rb != nil {
		return string(rb)
	}

	return ""
}

func GenRandomBase64String(l int) string {
	return base64.StdEncoding.EncodeToString(GenRandomBytes(l))
}

func GenRandomHexString(l int) string {
	return hex.EncodeToString(GenRandomBytes(l))
}

func genSessionID(l ...int) []byte {
	var size = sessEnthropy
	if len(l) > 0 && l[0] > 0 && l[0] < 1035 {
		size = l[0]
	}

	var h = sessHash()
	h.Write(GenRandomBytes(size))
	h.Write([]byte(strconv.FormatInt(time.Now().Unix(), 10)))
	id := make([]byte, 0, h.Size())
	id = h.Sum(id)
	return id
}

func GenSessionID(l ...int) string {
	return base64.StdEncoding.EncodeToString(genSessionID(l...))
}

func GenSessionIDBase32(l ...int) string {
	return base32.StdEncoding.EncodeToString(genSessionID(l...))
}
