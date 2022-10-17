package relay

import (
	"sync"
)

var HubRelayChan HubRelay
var once sync.Once

type HubRelay struct {
	IsClose chan struct{}
}

func InitHubRelay() {
	once.Do(func() {
		HubRelayChan = HubRelay{
			IsClose: make(chan struct{}),
		}
	})
}
