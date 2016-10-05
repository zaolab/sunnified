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

var ErrResourceNotFound = errors.New("resource not found")
var contentparser = make(map[string]ContentParser)

const (
	ReqmethodXMethodName = "X-HTTP-Method-Override"
	HTTPXForwardedFor = "X-Forwarded-For"
	HTTPXRealIP = "X-Real-IP"
	HTTPXRequestedWith = "X-Requested-With"
)

const (
	UserAnonymous int = iota
	UserUser
	UserPremiumUser
	UserWriter
	UserSuperWriter
	UserModerator
	UserSuperModerator
	UserAdmin
	UserSuperAdmin
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

func (r Redirection) URL() string {
	return r.url
}

func (r Redirection) Code() int {
	return r.code
}

type RedirectError struct {
	url string
}

func (re RedirectError) Error() string {
	return "Error redirecting service to " + re.url + "."
}

func (re RedirectError) URL() string {
	return re.url
}

func (re RedirectError) Code() int {
	return 500
}

type ExpectationError struct {
	size    int64
	maxsize int64
}

func (ee ExpectationError) Error() string {
	return fmt.Sprintf("request exceeds max file size of %dMB", ee.maxsize/1024/1024)
}

func (ee ExpectationError) IncomingSize() int64 {
	return ee.size
}

func (ee ExpectationError) MaxFileSize() int64 {
	return ee.maxsize
}

func (ee ExpectationError) Code() int {
	return http.StatusExpectationFailed
}

type AppError struct {
	code int
	err  string
}

func (ae AppError) Error() string {
	return ae.err
}

func (ae AppError) Code() int {
	return ae.code
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

func (p *parseState) Started() bool {
	return atomic.LoadInt32(&p.state) != 0
}

func (p *parseState) Ended() bool {
	return atomic.LoadInt32(&p.state) == 2
}

func (p *parseState) Start() (ok bool) {
	if atomic.CompareAndSwapInt32(&p.state, 0, 1) {
		p.Status.Add(1)
		ok = true
	}
	return
}

func (p *parseState) End(err error) {
	if atomic.CompareAndSwapInt32(&p.state, 1, 2) {
		p.Error = err
		p.Status.Done()
	}
}
