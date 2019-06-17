package worker

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
)

type fakeDB map[string][]BrokerPayload

var globalLock sync.RWMutex

func (f fakeDB) add(dsn string, payload BrokerPayload) {
	globalLock.Lock()
	defer globalLock.Unlock()
	f[dsn] = append(f[dsn], payload)
}
func (f fakeDB) last(dsn string) BrokerPayload {
	globalLock.RLock()
	defer globalLock.RUnlock()
	if len(f[dsn]) == 0 {
		return BrokerPayload{}
	}
	payload := f[dsn][len(f[dsn])-1]
	return payload
}
func (f fakeDB) all(dsn string) []BrokerPayload {
	globalLock.RLock()
	defer globalLock.RUnlock()
	var allPayload []BrokerPayload
	for _, payload := range f[dsn] {
		allPayload = append(allPayload, payload)
	}
	return allPayload
}

var fakedb fakeDB

func TestBroker(t *testing.T) {
	client := NewMQTTClient()
	if client == nil {
		os.Exit(1)
	}
	defer client.Close()

	fakedb = make(fakeDB)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)
	<-signalCh
}
