package sec

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"net/http"
	"strconv"
	"time"
)

const (
	//d_securityKey		    = "\x69\x51\xe8\x41\x50\x83\x19\xa4\xf0\x2f\xac\x7d\x99\xb7\x5e\xbe\x7e\x32\xf5\xa5\xf7\x1f\x43\x04\x96\xdd\x1b\xf0\x93\x4e\xc5\x44"
	//d_csrfToken			= "\xc7\x58\xa7\xf2\x15\x79\x54\x34\x24\xeb\x45\x50\x33\x0f\xa5\x52\x95\x36\x06\xb0\xb7\xdb\x5d\xa7\x07\xcf\xa5\x1c\x10\xe7\x4b\xd4"
	//d_hashSalt			= "\x5d\xfb\xcf\x47\x30\xce\x2e\x43\xfa\x1c\x5f\xee\x76\x0f\xd7\x31\x14\x07\x24\xa8\xbf\xd0\x3c\x88\xfc\xa3\xdc\x3b\xae\xaa\x3a\x15"
	//d_csrfTokenLife		= 14400
	CSRF_TOKEN_MIN_LIFE       = 3600
	CSRF_DEFAULT_TOKEN_LIFE   = 14400
	CSRF_DEFAULT_COOKIE_NAME  = "XSRF-TOKEN"
	CSRF_DEFAULT_REQUEST_NAME = "X-XSRF-TOKEN"
	CSRF_TIMESTAMP_LEN        = 5
	CSRF_RAND_TOKEN_LEN       = 16
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
		settings.Tokenlife = CSRF_DEFAULT_TOKEN_LIFE
	}
	if settings.Reqname == "" {
		settings.Reqname = CSRF_DEFAULT_REQUEST_NAME
	}
	if settings.Cookiename == "" {
		settings.Cookiename = CSRF_DEFAULT_COOKIE_NAME
	}

	return &CSRFGate{config: settings}
}

type CsrfRequestBody struct {
	Name   string
	Value  string
	Cookie *http.Cookie
	Ok     bool
}

// SetCSRFToken returns a CsrfRequestBody containing the name and value to be used
// as a query string or form input that can be verified by VerifyCSRFToken.
// Additionally, a cookie will be set (if ResponseWriter is not nil) to cross authenticate validity of token data if non exists
func (this *CSRFGate) CSRFToken(w http.ResponseWriter, r *http.Request) (crb CsrfRequestBody) {
	var (
		randToken   []byte
		msg         []byte
		writeCookie = false
		tstamp      = time.Now().Unix()
		// the current rolling global token.
		// this token is the share for the entire application
		// it rolls over to a new token every "csrf-token-life"
		currentToken = this.csrfCurrentToken(tstamp)
		ckie, err    = r.Cookie(this.config.Cookiename)
	)

	// gets the cookie containing the random token generated
	// the random token will be shared for all requests from the same machine/browser
	// this is a very simple mechanism for unique user identification
	if err == nil {
		randToken, err = AesCtrDecryptBase64(this.config.Key, ckie.Value)
	}
	// if there are no random token from the cookie,
	// generate a new one ourselves.
	if err != nil || len(randToken) != CSRF_RAND_TOKEN_LEN {
		randToken = GenRandomBytes(CSRF_RAND_TOKEN_LEN)

		if randToken == nil {
			// the randomness of this token is not as critical to security
			lenToFill := CSRF_RAND_TOKEN_LEN
			msgToHash := []byte(strconv.FormatInt(tstamp, 10))
			msgToHash = append(msgToHash, currentToken...)
			randToken = make([]byte, 0, CSRF_RAND_TOKEN_LEN)

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

	msg = make([]byte, 0, CSRF_TIMESTAMP_LEN+CSRF_RAND_TOKEN_LEN+len(currentToken))
	buf := bytes.NewBuffer(msg)
	binary.Write(buf, binary.LittleEndian, tstamp)

	// the csrf token consists of timestamp(5 bytes),
	// random bytes(16 bytes),
	// rolling global token(20bytes [sha1 checksum])
	msg = append(msg[0:CSRF_TIMESTAMP_LEN], randToken...)
	msg = append(msg, currentToken...)

	if value, err := AesCtrEncryptBase64(this.config.Key, msg); err == nil {
		if writeCookie {
			enc, err := AesCtrEncryptBase64(this.config.Key, randToken)

			if err != nil {
				return
			}

			ckie = &http.Cookie{
				Name:  this.config.Cookiename,
				Value: enc,
				Path:  "/",
			}
			if w != nil {
				http.SetCookie(w, ckie)
			}
		}

		crb.Name = this.config.Reqname
		crb.Value = value
		crb.Cookie = ckie
		crb.Ok = true
	}

	return
}

// VerifyCSRFToken checks whether the request r includes a valid CSRF token
func (this *CSRFGate) VerifyCSRFToken(r *http.Request) (valid bool) {
	var token string

	if token = r.Header.Get(this.config.Reqname); token != "" {
		// TODO: for cross domain, the request will first perform an OPTIONS
		// with Access-Control-Request-Headers: X-XSRF-TOKEN
		// we gotten respond with Access-Control-Allow-Headers: X-XSRF-TOKEN somehow
		// if router doesn't respond by mirroring the request
		if ckie, err := r.Cookie(this.config.Cookiename); err == nil {
			valid = token == ckie.Value
		}
	} else {
		r.ParseForm()
		token = r.Form.Get(this.config.Reqname)

		if token == "" {
			return
		}

		result, err := AesCtrDecryptBase64(this.config.Key, token)

		if err != nil || len(result) <= (CSRF_TIMESTAMP_LEN+CSRF_RAND_TOKEN_LEN) {
			return
		}

		lenTNC := CSRF_TIMESTAMP_LEN + CSRF_RAND_TOKEN_LEN

		tcreatedcap := CSRF_TIMESTAMP_LEN
		if tcreatedcap < 8 {
			tcreatedcap = 8
		}

		tcreated := make([]byte, CSRF_TIMESTAMP_LEN, tcreatedcap)

		// copy into a new slice, append overwrites original slice data
		copy(tcreated, result[0:CSRF_TIMESTAMP_LEN])
		ckietoken := make([]byte, CSRF_RAND_TOKEN_LEN)
		copy(ckietoken, result[CSRF_TIMESTAMP_LEN:lenTNC])
		reqtoken := make([]byte, len(result)-lenTNC)
		copy(reqtoken, result[lenTNC:])

		if CSRF_TIMESTAMP_LEN < 8 {
			filler := make([]byte, 8-CSRF_TIMESTAMP_LEN)
			tcreated = append(tcreated, filler...)
		}

		var tcreated64 int64
		binary.Read(bytes.NewBuffer(tcreated), binary.LittleEndian, &tcreated64)
		tstamp := time.Now().Unix()

		// check whether request token has already expired
		if (tcreated64+int64(this.config.Tokenlife)) < tstamp || tcreated64 > tstamp {
			return
		}

		// cookie authentication of csrf token is needed to ensure each machine has unique token
		if ckie, err := r.Cookie(this.config.Cookiename); err == nil {
			dec, err := AesCtrDecryptBase64(this.config.Key, ckie.Value)

			if err != nil || !bytes.Equal(dec, ckietoken) {
				return
			}
		} else {
			return
		}

		valid = bytes.Equal(reqtoken, this.csrfCurrentToken(tstamp)) || bytes.Equal(reqtoken, this.csrfPrevToken(tstamp))
	}

	return
}

func (this *CSRFGate) csrfCurrentToken(t ...int64) []byte {
	var tnow int64

	if len(t) > 0 {
		tnow = t[0]
	} else {
		tnow = time.Now().Unix()
	}
	iteration := tnow / int64(this.config.Tokenlife)
	return this.csrfIterToken(iteration)
}

func (this *CSRFGate) csrfCurrentTokenString(t ...int64) string {
	return string(this.csrfCurrentToken(t...))
}

func (this *CSRFGate) csrfPrevToken(t ...int64) []byte {
	var tnow int64

	if len(t) > 0 {
		tnow = t[0]
	} else {
		tnow = time.Now().Unix()
	}

	iteration := tnow / int64(this.config.Tokenlife)
	return this.csrfIterToken(iteration - 1)
}

func (this *CSRFGate) csrfPrevTokenString(t ...int64) string {
	return string(this.csrfPrevToken(t...))
}

func (this *CSRFGate) csrfIterToken(iteration int64) []byte {
	itertoken := strconv.FormatInt(iteration, 10)
	h := hmac.New(sha1.New, this.config.Token)
	h.Write([]byte(itertoken))
	hash := make([]byte, 0, h.Size())
	hash = h.Sum(hash)
	return hash
}

func (this *CSRFGate) csrfIterTokenString(iteration int64) string {
	return string(this.csrfIterToken(iteration))
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
	} else {
		return ""
	}
}

func GenRandomBase64String(l int) string {
	return base64.StdEncoding.EncodeToString(GenRandomBytes(l))
}

func GenRandomHexString(l int) string {
	return hex.EncodeToString(GenRandomBytes(l))
}

func GenSessionId() string {
	var h hash.Hash = sessHash()
	h.Write(GenRandomBytes(sessEnthropy))
	h.Write([]byte(strconv.FormatInt(time.Now().Unix(), 10)))
	id := make([]byte, 0, h.Size())
	id = h.Sum(id)
	return base64.StdEncoding.EncodeToString(id)
}
