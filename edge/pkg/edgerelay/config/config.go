package config

import (
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util"
	"sync"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeRelay
	relayID string
	// 本机ID
	nodeID string
	data   common.RelayData
}

func InitConfig(er *v1alpha1.EdgeRelay) {
	once.Do(func() {
		Config = Configure{
			EdgeRelay: *er,
			relayID:   "",
			nodeID:    util.GetHostname(),
		}
	})
}
func (config *Configure) SetRelayID(relayID string) {
	config.relayID = relayID
}
func (config *Configure) GetRelayID() string {
	return config.relayID
}
func (config *Configure) SetData(data common.RelayData) {
	config.data = data
}
func (config *Configure) GetData() common.RelayData {
	return config.data
}
