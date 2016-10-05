package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zaolab/sunnified/sec"
)

const (
	DefWindowSize     = 100
	DefTimeWindowSize = 2
	DefInterval       = 30
	DefDigits         = 6
)

var (
	ErrContainsColon = errors.New("issuer/account name cannot contain colon")
	ErrInvalidLen    = errors.New("password can only be 6 or 8 characters long")
)

type OTP interface {
	Type() string
	SetIssuer(string) error
	Issuer() string
	SetAccountName(string) error
	AccountName() string
	SetHashFunc(string, func() hash.Hash)
	UseSHA1()
	UseSHA256()
	UseSHA512()
	HashFuncName() string
	SetDigits(int) error
	Digits() int
	SetCounter(uint64)
	Counter() uint64
	IncCounter()
	SetWindowSize(int)
	WindowSize() int
	VerifyAt(string, uint64) bool
	Verify(string) bool
	String() string
	URI() string
	PasswordAt(uint64) string
	Password() string
	SetInterval(int)
	Interval() int
}

type HOTP struct {
	issuer  string
	account string
	h       hash.Hash
	hname   string
	c       []byte
	secret  []byte
	counter uint64
	window  int
	digits  int
}

func (ho *HOTP) SetHashFunc(name string, h func() hash.Hash) {
	ho.hname = name
	ho.h = hmac.New(h, ho.secret)
}

func (ho *HOTP) UseSHA1() {
	if ho.hname != "SHA1" {
		ho.hname = "SHA1"
		ho.h = hmac.New(sha1.New, ho.secret)
	}
}

func (ho *HOTP) UseSHA256() {
	if ho.hname != "SHA256" {
		ho.hname = "SHA256"
		ho.h = hmac.New(sha256.New, ho.secret)
	}
}

func (ho *HOTP) UseSHA512() {
	if ho.hname != "SHA512" {
		ho.hname = "SHA512"
		ho.h = hmac.New(sha512.New, ho.secret)
	}
}

func (ho *HOTP) HashFuncName() string {
	return ho.hname
}

func (ho *HOTP) SetIssuer(issuer string) error {
	if strings.Contains(issuer, ":") {
		return ErrContainsColon
	}
	ho.issuer = issuer
	return nil
}

func (ho *HOTP) Issuer() string {
	return ho.issuer
}

func (ho *HOTP) SetAccountName(account string) error {
	if strings.Contains(account, ":") {
		return ErrContainsColon
	}
	ho.account = account
	return nil
}

func (ho *HOTP) AccountName() string {
	return ho.account
}

func (ho *HOTP) SetDigits(d int) error {
	if d == 6 || d == 8 {
		ho.digits = d
		return nil
	}
	return ErrInvalidLen
}

func (ho *HOTP) Digits() int {
	if ho.digits == 0 {
		return DefDigits
	}
	return ho.digits
}

func (ho *HOTP) SetCounter(c uint64) {
	ho.counter = c
}

func (ho *HOTP) IncCounter() {
	ho.counter++
}

func (ho *HOTP) Counter() uint64 {
	return ho.counter
}

func (ho *HOTP) SetWindowSize(c int) {
	ho.window = c
}

func (ho *HOTP) WindowSize() int {
	return ho.window
}

func (ho *HOTP) VerifyAt(password string, counter uint64) bool {
	return ho.verifyOTP(password, counter, 0) != -1
}

func (ho *HOTP) Verify(password string) bool {
	if count := ho.verifyOTP(password, ho.counter, ho.window); count != -1 {
		ho.counter += uint64(count) + 1
		return true
	}
	return false
}

func (ho *HOTP) String() string {
	return base32.StdEncoding.EncodeToString(ho.secret)
}

func (ho *HOTP) URI() string {
	return ho.genURI(ho.Type(), 0)
}

func (ho *HOTP) PasswordAt(counter uint64) string {
	binary.BigEndian.PutUint64(ho.c, counter)

	ho.h.Reset()
	ho.h.Write(ho.c)
	ha := make([]byte, 0, ho.h.Size())
	ha = ho.h.Sum(ha)

	offset := binary.BigEndian.Uint16(ha[18:20]) & 0xf
	password32 := binary.BigEndian.Uint32(ha[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%010d", password32)[10-ho.Digits() : 10]
}

func (ho *HOTP) Password() string {
	return ho.PasswordAt(ho.counter)
}

func (ho *HOTP) Type() string {
	return "hotp"
}

func (ho *HOTP) genURI(otptype string, interval int) string {
	accountname := strings.Replace(url.QueryEscape(ho.account), "+", "%20", -1)
	params := url.Values{}
	params.Add("secret", ho.String())
	params.Add("algorithm", ho.hname)

	if otptype == "hotp" {
		params.Add("counter", strconv.FormatInt(int64(ho.counter), 10))
	} else if otptype == "totp" {
		params.Add("period", strconv.Itoa(interval))
	}

	if ho.issuer != "" {
		params.Add("issuer", ho.issuer)
		issuer := strings.Replace(url.QueryEscape(ho.issuer), "+", "%20", -1)

		return fmt.Sprintf(
			"otpauth://%s/%s:%s?%s",
			otptype,
			issuer,
			accountname,
			params.Encode(),
		)
	}

	return fmt.Sprintf(
		"otpauth://%s/%s?%s",
		otptype,
		accountname,
		params.Encode(),
	)
}

func (ho *HOTP) verifyOTP(password string, counter uint64, window int) int {
	if len(password) != ho.Digits() {
		return -1
	}

	maxcount := uint64(window + 1)
	bpassword := []byte(password)

	for i := uint64(0); i < maxcount; i++ {
		ourpassword := []byte(ho.PasswordAt(counter + i))
		if subtle.ConstantTimeCompare(ourpassword, bpassword) == 1 {
			return int(i)
		}
	}

	return -1
}

func (ho *HOTP) SetInterval(_ int) {}

func (ho *HOTP) Interval() int {
	return 0
}

type TOTP struct {
	*HOTP
	interval uint64
}

func (to *TOTP) SetInterval(seconds int) {
	to.interval = uint64(seconds)
}

func (to *TOTP) Interval() int {
	return int(to.interval)
}

func (to *TOTP) Verify(password string) bool {
	counter := uint64(time.Now().Unix()) / to.interval
	ok := to.HOTP.VerifyAt(password, counter)

	if !ok && to.window > 0 {
		window := uint64(to.window)

		for i := uint64(0); i < window; i++ {
			ok = to.HOTP.VerifyAt(password, counter+i+1) || to.HOTP.VerifyAt(password, counter-i-1)
			if ok {
				break
			}
		}
	}

	return ok
}

func (to *TOTP) VerifyAt(password string, time uint64) bool {
	return to.HOTP.VerifyAt(password, time/to.interval)
}

func (to *TOTP) URI() string {
	return to.genURI(to.Type(), int(to.interval))
}

func (to *TOTP) Password() string {
	return to.HOTP.PasswordAt(uint64(time.Now().Unix()) / to.interval)
}

func (to *TOTP) PasswordAt(time uint64) string {
	return to.HOTP.PasswordAt(time / to.interval)
}

func (to *TOTP) Type() string {
	return "totp"
}

func NewHOTP(secret string) *HOTP {
	if secret, err := base32.StdEncoding.DecodeString(secret); err == nil {
		return &HOTP{
			h:      hmac.New(sha1.New, secret),
			hname:  "SHA1",
			c:      make([]byte, 8),
			secret: secret,
			window: DefWindowSize,
			digits: DefDigits,
		}
	}

	return nil
}

func NewHOTPAccount(secret, issuer, account string) (hotp *HOTP) {
	hotp = NewHOTP(secret)

	if hotp != nil {
		hotp.issuer = issuer
		hotp.account = account
	}

	return
}

func NewTOTP(secret string) *TOTP {
	if secret, err := base32.StdEncoding.DecodeString(secret); err == nil {
		return &TOTP{
			HOTP: &HOTP{
				h:      hmac.New(sha1.New, secret),
				hname:  "SHA1",
				c:      make([]byte, 8),
				secret: secret,
				window: DefTimeWindowSize,
				digits: DefDigits,
			},
			interval: DefInterval,
		}
	}

	return nil
}

func NewTOTPAccount(secret, issuer, account string) (totp *TOTP) {
	totp = NewTOTP(secret)

	if totp != nil {
		totp.issuer = issuer
		totp.account = account
	}

	return
}

func NewOTP(uri string) (otp OTP) {
	if u, err := url.Parse(uri); err == nil {
		var (
			params  = u.Query()
			secret  = params.Get("secret")
			issuer  string
			account string
		)

		switch strings.ToLower(u.Host) {
		case "totp":
			otp = NewTOTP(secret)

			if period := params.Get("period"); period != "" {
				if i, err := strconv.Atoi(period); err == nil {
					otp.SetInterval(i)
				}
			}
		case "hotp":
			otp = NewHOTP(secret)

			if counter := params.Get("counter"); counter != "" {
				if i, err := strconv.Atoi(counter); err == nil {
					otp.SetCounter(uint64(i))
				}
			}
		}

		switch strings.ToUpper(params.Get("algorithm")) {
		case "SHA256":
			otp.SetHashFunc("SHA256", sha256.New)
		case "SHA512":
			otp.SetHashFunc("SHA512", sha512.New)
		}

		if digits := params.Get("digits"); digits != "" {
			if i, err := strconv.Atoi(digits); err == nil {
				otp.SetDigits(i)
			}
		}

		issuer = params.Get("issuer")

		if label, err := url.QueryUnescape(strings.TrimPrefix(u.Path, "/")); err == nil {
			if strings.Contains(label, ":") {
				arr := strings.SplitN(label, ":", 2)
				issuer = arr[0]
				account = arr[1]
			} else {
				account = label
			}
		}

		otp.SetIssuer(issuer)
		otp.SetAccountName(account)
	}

	return
}

func GenerateSecret() string {
	return base32.StdEncoding.EncodeToString(sec.GenRandomBytes(10))
}
