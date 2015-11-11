package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrResourceNotFound = errors.New("Resource not found")
var contentparser = make(map[string]ContentParser)

const (
	REQMETHOD_X_METHOD_NAME = "X-HTTP-Method-Override"
	HTTP_X_FORWARDED_FOR    = "X-Forwarded-For"
	HTTP_X_REAL_IP          = "X-Real-IP"
	HTTP_X_REQUESTED_WITH   = "X-Requested-With"
)

const (
	USER_ANONYMOUS int = iota
	USER_USER
	USER_PREMIUMUSER
	USER_WRITER
	USER_SUPERWRITER
	USER_MODERATOR
	USER_SUPERMODERATOR
	USER_ADMIN
	USER_SUPERADMIN
)

type Q map[string]string

type SessionManager interface {
	ID() string
	String(string) string
	Int(string) int
	Int64(string) int64
	Float32(string) float32
	Float64(string) float64
	Bool(string) bool
	Byte(string) byte
	Get(string) interface{}
	MapValue(string, interface{})
	IPAddress() string
	UserAgent() string
	Created() time.Time
	Accessed() time.Time
	Expiry() time.Time
	AuthUser() UserModel
	Set(string, interface{})
	Remove(string)
	SetIPAddress(string)
	SetUserAgent(string)
	SetExpiry(time.Time)
	SetAuthUser(UserModel)
	SetAuthUserData(id, email, name string, lvl int)
	SetAnonymous()
	IsAuthUser(id string) bool
	UpdateAccessed()
	AddFlash(string)
	HasFlash() bool
	Flash() string
	AllFlashes() []string
	PeekFlashes() []string
	LenFlashes() int
}

type UserModel interface {
	ID() string
	Email() string
	Name() string
	Level() int
	IsSuperAdmin() bool
	IsAdmin() bool
	IsSuperModerator() bool
	IsModerator() bool
	IsSuperWriter() bool
	IsWriter() bool
	IsPremiumUser() bool
	IsUser() bool
	IsAnonymous() bool
}

type CacheManager interface {
	Get(string) interface{}
	MapValue(string, interface{})
	Set(string, interface{}, time.Duration)
	Delete(string)
	Clear()
}

type ContextError interface {
	Error() string
	Code() int
}

type ContextHandler interface {
	ServeContextHTTP(*Context)
}

type ContextOptionsHandler interface {
	ServeContextOptions(*Context, map[string]string)
}

type Redirection struct {
	code int
	url  string
}

func (this Redirection) URL() string {
	return this.url
}

func (this Redirection) Code() int {
	return this.code
}

type RedirectError struct {
	url string
}

func (this RedirectError) Error() string {
	return "Error redirecting service to " + this.url + "."
}

func (this RedirectError) URL() string {
	return this.url
}

func (this RedirectError) Code() int {
	return 500
}

type ExpectationError struct {
	size    int64
	maxsize int64
}

func (this ExpectationError) Error() string {
	return fmt.Sprintf("Request exceeds max file size of %dMB", this.maxsize/1024/1024)
}

func (this ExpectationError) IncomingSize() int64 {
	return this.size
}

func (this ExpectationError) MaxFileSize() int64 {
	return this.maxsize
}

func (this ExpectationError) Code() int {
	return http.StatusExpectationFailed
}

type AppError struct {
	code int
	err  string
}

func (this AppError) Error() string {
	return this.err
}

func (this AppError) Code() int {
	return this.code
}

type ContentParser func(io.Reader) map[string]interface{}

func SetContentParser(ctype string, f ContentParser) {
	contentparser[strings.ToLower(ctype)] = f
}

func GetContentParser(ctype string) ContentParser {
	return contentparser[ctype]
}

type parseState struct {
	state  int32
	Status sync.WaitGroup
	Error  error
}

func (this *parseState) Started() bool {
	return atomic.LoadInt32(&this.state) != 0
}

func (this *parseState) Ended() bool {
	return atomic.LoadInt32(&this.state) == 2
}

func (this *parseState) Start() (ok bool) {
	if atomic.CompareAndSwapInt32(&this.state, 0, 1) {
		this.Status.Add(1)
		ok = true
	}
	return
}

func (this *parseState) End(err error) {
	if atomic.CompareAndSwapInt32(&this.state, 1, 2) {
		this.Error = err
		this.Status.Done()
	}
}
