package cloudrelay

import (
	"context"
	"encoding/json"
	"fmt"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/viaduct/pkg/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sync"
)

var once sync.Once

type CloudRelay struct {
	enable  bool
	status  bool
	relayID string
	// kubeClient    kubernetes.Interface
	crdClient crdClientset.Interface
}

var RelayHandle *CloudRelay

func InitCloudRelay() {
	// init的后去k8s的config查询有没有存储下来的中继数据，这部分在server.go
	once.Do(func() {
		RelayHandle = &CloudRelay{
			enable:    true,
			status:    true,
			relayID:   "",
			crdClient: client.GetCRDClient(),
		}
	})
}

func (relayHandle *CloudRelay) LoadRelayID() {
	myRelayRCs, err := relayHandle.crdClient.RelaysV1().Relayrcs(constants.DefaultNameSpace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Failed to get relay message from relaycontroller", err)
	}
	if len(myRelayRCs.Items) == 0 {
		return
	}
	myRelayRC := myRelayRCs.Items[0]
	relayHandle.relayID = myRelayRC.Name
}

func (relayHandle *CloudRelay) IsRelayIdEmpty() bool {
	if relayHandle.relayID == "" {
		return true
	}
	return false
}

func (relayHandle *CloudRelay) ChangeDesToRelay(msg *beehiveModel.Message) (string, *beehiveModel.Message) {

	oldID, rmsg, err := relayHandle.SealMessage(msg)
	if err != nil {
		fmt.Errorf("ChangeDesToRelay failed")
		return oldID, msg
	}
	return oldID, rmsg
}

func (relayHandle *CloudRelay) SealMessage(msg *beehiveModel.Message) (string, *beehiveModel.Message, error) {
	nodeID := relayHandle.relayID
	oldID, resource, err := messagelayer.BuildResource(nodeID, msg.Router.Resource)
	if err != nil {
		return oldID, msg, fmt.Errorf("build relay node resource failed")
	}
	relayMsg := msg.Clone(msg)

	relayMsg.Header.ID = msg.Header.ID
	relayMsg.Router.Resource = resource
	relayMsg.Router.Group = constants.RelayGroupName

	//contentMsg, err := json.Marshal(msg)
	if err != nil {
		klog.V(4).Infof("RelayHandleServer Umarshal failed", err)
	}
	relayMsg.Content = msg
	return oldID, relayMsg, nil
}

func (relayHandle *CloudRelay) UnsealMessage(container *mux.MessageContainer) *mux.MessageContainer {
	var rcontainer *mux.MessageContainer

	content, err := container.Message.GetContentData()
	if err != nil {
		return nil
	}
	err = json.Unmarshal(content, &rcontainer)
	if err != nil {
		klog.Infof("UnsealMessage Unmarshal failed:%v", err)
		return nil
	}

	return rcontainer
}

func (relayHandle *CloudRelay) GetRelayId() string {
	return relayHandle.relayID
}
func (relayHandle *CloudRelay) SetRelayId(relayID string) {
	relayHandle.relayID = relayID
}
func (relayHandle *CloudRelay) GetStatus() bool {
	return relayHandle.status
}
func (relayHandle *CloudRelay) SetStatus(status bool) {
	relayHandle.status = status
}
