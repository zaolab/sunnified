package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"github.com/zaolab/sunnified/sec"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
	"errors"
)

const (
	DEFAULT_WINDOW_SIZE      = 100
	DEFAULT_TIME_WINDOW_SIZE = 2
	DEFAULT_INTERVAL         = 30
	DEFAULT_DIGITS           = 6
)

var (
	ErrContainsColon = errors.New("Issuer/Account name cannot contain colon")
	ErrInvalidLen = errors.New("Password can only be 6 or 8 characters long")
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

func (this *HOTP) SetHashFunc(name string, h func() hash.Hash) {
	this.hname = name
	this.h = hmac.New(h, this.secret)
}

func (this *HOTP) UseSHA1() {
	if this.hname != "SHA1" {
		this.hname = "SHA1"
		this.h = hmac.New(sha1.New, this.secret)
	}
}

func (this *HOTP) UseSHA256() {
	if this.hname != "SHA256" {
		this.hname = "SHA256"
		this.h = hmac.New(sha256.New, this.secret)
	}
}

func (this *HOTP) UseSHA512() {
	if this.hname != "SHA512" {
		this.hname = "SHA512"
		this.h = hmac.New(sha512.New, this.secret)
	}
}

func (this *HOTP) HashFuncName() string {
	return this.hname
}

func (this *HOTP) SetIssuer(issuer string) error {
	if strings.Contains(issuer, ":") {
		return ErrContainsColon
	}
	this.issuer = issuer
	return nil
}

func (this *HOTP) Issuer() string {
	return this.issuer
}

func (this *HOTP) SetAccountName(account string) error {
	if strings.Contains(account, ":") {
		return ErrContainsColon
	}
	this.account = account
	return nil
}

func (this *HOTP) AccountName() string {
	return this.account
}

func (this *HOTP) SetDigits(d int) error {
	if d == 6 || d == 8 {
		this.digits = d
		return nil
	}
	return ErrInvalidLen
}

func (this *HOTP) Digits() int {
	if this.digits == 0 {
		return DEFAULT_DIGITS
	}
	return this.digits
}

func (this *HOTP) SetCounter(c uint64) {
	this.counter = c
}

func (this *HOTP) IncCounter() {
	this.counter++
}

func (this *HOTP) Counter() uint64 {
	return this.counter
}

func (this *HOTP) SetWindowSize(c int) {
	this.window = c
}

func (this *HOTP) WindowSize() int {
	return this.window
}

func (this *HOTP) VerifyAt(password string, counter uint64) bool {
	return this.verifyOTP(password, counter, 0) != -1
}

func (this *HOTP) Verify(password string) bool {
	if count := this.verifyOTP(password, this.counter, this.window); count != -1 {
		this.counter += uint64(count) + 1
		return true
	}
	return false
}

func (this *HOTP) String() string {
	return base32.StdEncoding.EncodeToString(this.secret)
}

func (this *HOTP) URI() string {
	return this.genURI(this.Type(), 0)
}

func (this *HOTP) PasswordAt(counter uint64) string {
	binary.BigEndian.PutUint64(this.c, counter)

	this.h.Reset()
	this.h.Write(this.c)
	ha := make([]byte, 0, this.h.Size())
	ha = this.h.Sum(ha)

	offset := binary.BigEndian.Uint16(ha[18:20]) & 0xf
	password32 := binary.BigEndian.Uint32(ha[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%010d", password32)[10-this.Digits() : 10]
}

func (this *HOTP) Password() string {
	return this.PasswordAt(this.counter)
}

func (this *HOTP) Type() string {
	return "hotp"
}

func (this *HOTP) genURI(otptype string, interval int) string {
	accountname := strings.Replace(url.QueryEscape(this.account), "+", "%20", -1)
	params := url.Values{}
	params.Add("secret", this.String())
	params.Add("algorithm", this.hname)

	if otptype == "hotp" {
		params.Add("counter", strconv.FormatInt(int64(this.counter), 10))
	} else if otptype == "totp" {
		params.Add("period", strconv.Itoa(interval))
	}

	if this.issuer != "" {
		params.Add("issuer", this.issuer)
		issuer := strings.Replace(url.QueryEscape(this.issuer), "+", "%20", -1)

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

func (this *HOTP) verifyOTP(password string, counter uint64, window int) int {
	if len(password) != this.Digits() {
		return -1
	}

	maxcount := uint64(window + 1)
	bpassword := []byte(password)

	for i := uint64(0); i < maxcount; i++ {
		ourpassword := []byte(this.PasswordAt(counter + i))
		if subtle.ConstantTimeCompare(ourpassword, bpassword) == 1 {
			return int(i)
		}
	}

	return -1
}

func (this *HOTP) SetInterval(_ int) {}

func (this *HOTP) Interval() int {
	return 0
}

type TOTP struct {
	*HOTP
	interval uint64
}

func (this *TOTP) SetInterval(seconds int) {
	this.interval = uint64(seconds)
}

func (this *TOTP) Interval() int {
	return int(this.interval)
}

func (this *TOTP) Verify(password string) bool {
	counter := uint64(time.Now().Unix()) / this.interval
	ok := this.HOTP.VerifyAt(password, counter)

	if !ok && this.window > 0 {
		window := uint64(this.window)

		for i := uint64(0); i < window; i++ {
			ok = this.HOTP.VerifyAt(password, counter+i+1) || this.HOTP.VerifyAt(password, counter-i-1)
			if ok {
				break
			}
		}
	}

	return ok
}

func (this *TOTP) VerifyAt(password string, time uint64) bool {
	return this.HOTP.VerifyAt(password, time/this.interval)
}

func (this *TOTP) URI() string {
	return this.genURI(this.Type(), int(this.interval))
}

func (this *TOTP) Password() string {
	return this.HOTP.PasswordAt(uint64(time.Now().Unix()) / this.interval)
}

func (this *TOTP) PasswordAt(time uint64) string {
	return this.HOTP.PasswordAt(time / this.interval)
}

func (this *TOTP) Type() string {
	return "totp"
}

func NewHOTP(secret string) *HOTP {
	if secret, err := base32.StdEncoding.DecodeString(secret); err == nil {
		return &HOTP{
			h:      hmac.New(sha1.New, secret),
			hname:  "SHA1",
			c:      make([]byte, 8),
			secret: secret,
			window: DEFAULT_WINDOW_SIZE,
			digits: DEFAULT_DIGITS,
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
				window: DEFAULT_TIME_WINDOW_SIZE,
				digits: DEFAULT_DIGITS,
			},
			interval: DEFAULT_INTERVAL,
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
