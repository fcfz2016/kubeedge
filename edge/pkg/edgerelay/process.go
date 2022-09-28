package edgerelay

import (
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"k8s.io/klog/v2"
	"time"
)

type NodeAddress struct {
	// Required
	IP string `json:"ip,omitempty"`
	// Required
	Port int64 `json:"port,omitempty"`
}
type RelayData struct {
	//
	AddrData map[string]NodeAddress `json:"addrdata,omitempty"`
}

func (er *EdgeRelay) SaveRelayID(relayID string) {

	// 更新config和数据库
	config.Config.SetRelayID(relayID)
	// 判断数据库中能不能查到，不能查到就insert，能查到就update
	meta := &dao.Meta{
		Key:   constants.RelayID,
		Type:  constants.RelayType,
		Value: string(relayID)}
	err := dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("save relayId failed", err)
		return
	}
}
func (er *EdgeRelay) LoadRelayID() string {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayID)
	if err != nil {
		klog.Errorf("query relayID failed")
	}
	if metas == nil {
		return ""
	}
	var result = *metas
	return result[0]
}
func (er *EdgeRelay) MsgFromEdgeHub() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeRelay MsgFromEdgeHub stop")
			return
		default:
		}
		// ch <- message
		message, err := beehiveContext.Receive(modules.EdgeRelayModuleName)
		if err != nil {
			klog.Errorf("edgerelay failed to receive message from edgehub: %v", err)
			time.Sleep(time.Second)
		}
		// 调用HandleMsgFromEdgeHub
		//er.HandleMsgFromEdgeHub(&message)
	}

}
