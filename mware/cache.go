package mware

import "github.com/zaolab/sunnified/web"

const (
	_ = iota
	CACHE_PROFILE_NOCACHE
	CACHE_PROFILE_NOSTORE
	CACHE_PROFILE_PUBLIC
)

func NewCacheMiddleWare(profile int) CacheMiddleWare {
	return CacheMiddleWare{
		profile: profile,
	}
}

func CacheNoCacheMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CACHE_PROFILE_NOCACHE)
}

func CacheNoStoreMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CACHE_PROFILE_NOSTORE)
}

func CachePublicMiddleWareConstructor() MiddleWare {
	return NewCacheMiddleWare(CACHE_PROFILE_PUBLIC)
}

type CacheMiddleWare struct {
	BaseMiddleWare
	profile int
}

func (this CacheMiddleWare) Request(ctxt *web.Context) {
	switch this.profile {
	case CACHE_PROFILE_NOCACHE:
		ctxt.PrivateNoCache()
	case CACHE_PROFILE_NOSTORE:
		ctxt.PrivateNoStore()
	case CACHE_PROFILE_PUBLIC:
		ctxt.PublicCache(0)
	}
}
