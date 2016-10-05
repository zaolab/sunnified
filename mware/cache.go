package mware

import "github.com/zaolab/sunnified/web"

const (
	_ = iota
	CacheProfileNoCache
	CacheProfileNoStore
	CacheProfilePublic
)

func NewCacheMiddleWare(profile int) CacheMiddleWare {
	return CacheMiddleWare{
		profile: profile,
	}
}

func CacheNoCacheMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CacheProfileNoCache)
}

func CacheNoStoreMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CacheProfileNoStore)
}

func CachePublicMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CacheProfilePublic)
}

type CacheMiddleWare struct {
	BaseMiddleWare
	profile int
}

func (mw CacheMiddleWare) Request(ctxt *web.Context) {
	switch mw.profile {
	case CacheProfileNoCache:
		ctxt.PrivateNoCache()
	case CacheProfileNoStore:
		ctxt.PrivateNoStore()
	case CacheProfilePublic:
		ctxt.PublicCache(0)
	}
}
