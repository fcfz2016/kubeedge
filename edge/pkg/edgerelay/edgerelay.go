package edgerelay

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"k8s.io/klog/v2"
)

type EdgeRelay struct {
	enable bool
}

var _ core.Module = (*EdgeRelay)(nil)

func Register(er *v1alpha1.EdgeRelay) {
	config.InitConfig(er)
	core.Register(newEdgeRelay(er.Enable))
}

func (er *EdgeRelay) Name() string {
	return modules.EdgeRelayModuleName
}

func (er *EdgeRelay) Group() string {
	return modules.RelayGroup
}

func (er *EdgeRelay) Enable() bool {
	return er.enable
}

func newEdgeRelay(enable bool) *EdgeRelay {

	return &EdgeRelay{
		enable: enable,
	}
}

func (er *EdgeRelay) Start() {
	klog.Info("Start edge relay")

	// 首先读取qlite里的信息，放到内存中（更新config）
	// er.Load()

	// 不需要开协程 er.MsgFromOtherEdge()
	er.server()
	go er.MsgFromEdgeHub()

}
