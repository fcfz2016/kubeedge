package relaycontroller

import (
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/relays/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"reflect"
)

const (
	RelayCloseOperation  = "closerelay"
	RelayOpenOperation   = "openrelay"
	RelayUpdateOperation = "updaterelay"

	GroupResource     = "relay"
	ResourceTypeRelay = "relayres"
)

// 方法的具体实现
func (rc *RelayController) checkRelay() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop checkRelay")
			return
		case e := <-rc.relayrcManager.Events():
			relayrc, ok := e.Object.(*v1.Relayrc)

			if !ok {
				klog.Warningf("Object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				rc.relayrcAdded(relayrc)
			case watch.Deleted:
				rc.relayrcDeleted(relayrc)
			case watch.Modified:
				rc.relayrcUpdated(relayrc)
			default:
				klog.Warningf("Device event type: %s unsupported", e.Type)
			}

		}

	}
}

func (rc *RelayController) relayrcAdded(relayrc *v1.Relayrc) {
	rc.relayrcManager.RelayInfo.Store(relayrc.Name, relayrc)
	klog.Warningf("Relay added", relayrc.Spec.RelayID)
	if relayrc.Spec.Open {
		if isRelayIDExist(relayrc.Spec.RelayID) {
			klog.Warningf("store RelayID")
			// 下发
			msg := buildControllerMessage(relayrc.Spec.RelayID, relayrc.Namespace, RelayOpenOperation, relayrc)
			err := rc.messageLayer.Send(*msg)
			klog.Warningf("relay added send error", err)
		} else {
			klog.Warningf("RelayID is empty")
		}
	}
}

func (rc *RelayController) relayrcDeleted(relayrc *v1.Relayrc) {
	rc.relayrcManager.RelayInfo.Delete(relayrc.Name)
	klog.Warningf("Relay delete")
	// 下发关闭信息
	msg := buildControllerMessage(relayrc.Spec.RelayID, relayrc.Namespace, RelayCloseOperation, relayrc)
	err := rc.messageLayer.Send(*msg)
	klog.Warningf("relay close send error", err)
}

func (rc *RelayController) relayrcUpdated(relayrc *v1.Relayrc) {
	// klog.Warningf("Relay updated", relayrc.Spec.RelayID)
	value, ok := rc.relayrcManager.RelayInfo.Load(relayrc.Name)
	rc.relayrcManager.RelayInfo.Store(relayrc.Name, relayrc)
	if ok {
		cacheRelayrc := value.(*v1.Relayrc)
		if isRelayRCUpdated(cacheRelayrc, relayrc) {
			// 如果是开关改动
			if isSwitchUpdated(cacheRelayrc.Spec.Open, relayrc.Spec.Open) {
				if relayrc.Spec.Open {
					if isRelayIDExist(relayrc.Spec.RelayID) {
						klog.Warningf("Relay updated", relayrc.Spec.RelayID)
						msg := buildControllerMessage(relayrc.Spec.RelayID, relayrc.Namespace, RelayOpenOperation, relayrc)
						err := rc.messageLayer.Send(*msg)
						if err != nil {
							klog.Warningf("relay open msg send error", err)
						}
					}
				} else {
					klog.Warningf("Relay updated", relayrc.Spec.RelayID)
					msg := buildControllerMessage(relayrc.Spec.RelayID, relayrc.Namespace, RelayCloseOperation, relayrc)
					err := rc.messageLayer.Send(*msg)
					if err != nil {
						klog.Warningf("relay close msg send error", err)
					}
				}
			} else if isRelayIDUpdated(cacheRelayrc.Spec.RelayID, relayrc.Spec.RelayID) {
				if relayrc.Spec.Open {
					klog.Warningf("Relay updated", relayrc.Spec.RelayID)
					// 新的relayID信息发送给旧的relayID处理
					msg := buildControllerMessage(cacheRelayrc.Spec.RelayID, relayrc.Namespace, RelayUpdateOperation, relayrc)
					err := rc.messageLayer.Send(*msg)
					if err != nil {
						klog.Warningf("relay update msg send error", err)
					}
				}
			} else {
				klog.Warningf("Relay updated", relayrc.Spec.RelayID)
				klog.Warningf("any other relay msg updated")
			}
		}
	}
}

func isRelayIDExist(id string) bool {
	if id != "" {
		return true
	}
	return false
}

func isSwitchUpdated(old bool, new bool) bool {
	return old != new
}

func isRelayIDUpdated(old string, new string) bool {
	return old != new
}

func isRelayRCUpdated(old *v1.Relayrc, new *v1.Relayrc) bool {
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Spec, new.Spec) || !reflect.DeepEqual(old.Status, new.Status)
}

func buildControllerMessage(nodeID, namespace, opr string, relayrc *v1.Relayrc) *model.Message {
	msg := model.NewMessage("")

	resource, err := messagelayer.BuildResource(nodeID, namespace, ResourceTypeRelay, "")
	if err != nil {
		klog.Warningf("Built message resource failed with error: %s", err)

		return nil
	}

	msg.BuildRouter(modules.RelayControllerModuleName, GroupResource, resource, opr)
	content := relayrc.Spec
	msg.Content = content

	klog.Warningf("relaycontroller send msg", msg.Router.Operation)
	return msg
}
