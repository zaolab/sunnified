// +build !windows,!darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package sunnified

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
)

var graceshut int32

func GracefulShutDown() {
	if atomic.CompareAndSwapInt32(&graceshut, 0, 1) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		go func() {
			<-c
			mutex.RLock()
			var serverslen = len(servers)
			var allservers = make([]*SunnyApp, serverslen)
			copy(allservers, servers)
			mutex.RUnlock()

			w := &sync.WaitGroup{}
			w.Add(serverslen)

			for _, server := range allservers {
				if !server.Close(func() { w.Done() }) {
					w.Done()
				}
			}

			w.Wait()
			os.Exit(0)
		}()
	}
}
