package config

import (
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/relays/v1"
	"github.com/kubeedge/kubeedge/pkg/util"
	"sync"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeRelay
	relayID string
	// 本机ID
	nodeID      string
	status      bool
	isRelayNode bool
	data        v1.RelayData
}

func InitConfig(er *v1alpha1.EdgeRelay) {
	once.Do(func() {
		Config = Configure{
			EdgeRelay:   *er,
			relayID:     "",
			status:      false,
			isRelayNode: false,
			nodeID:      util.GetHostname(),
		}
	})
}
func (config *Configure) SetRelayID(relayID string) {
	config.relayID = relayID
}
func (config *Configure) GetRelayID() string {
	return config.relayID
}
func (config *Configure) SetData(data v1.RelayData) {
	config.data = data
}
func (config *Configure) GetData() v1.RelayData {
	return config.data
}
func (config *Configure) GetNodeID() string {
	return config.nodeID
}
func (config *Configure) GetStatus() bool {
	return config.status
}
func (config *Configure) SetStatus(status bool) {
	config.status = status
}

func (config *Configure) SetIsRelayNode(isOr bool) {
	config.isRelayNode = isOr
}

func (config *Configure) GetIsRelayNode() bool {
	return config.isRelayNode
}
