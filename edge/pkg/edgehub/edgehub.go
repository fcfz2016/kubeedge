package edgehub

import (
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/relay"
	"sync"
	"time"

	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	relayConfig "github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var HasTLSTunnelCerts = make(chan bool, 1)

//EdgeHub defines edgehub object structure
type EdgeHub struct {
	certManager   certificate.CertManager
	chClient      clients.Adapter
	reconnectChan chan struct{}
	rateLimiter   flowcontrol.RateLimiter
	keeperLock    sync.RWMutex
	enable        bool
}

var _ core.Module = (*EdgeHub)(nil)

func newEdgeHub(enable bool) *EdgeHub {
	return &EdgeHub{
		enable:        enable,
		reconnectChan: make(chan struct{}),
		rateLimiter: flowcontrol.NewTokenBucketRateLimiter(
			float32(config.Config.EdgeHub.MessageQPS),
			int(config.Config.EdgeHub.MessageBurst)),
	}
}

// Register register edgehub
func Register(eh *v1alpha1.EdgeHub, nodeName string) {
	config.InitConfigure(eh, nodeName)
	relay.InitHubRelay()
	core.Register(newEdgeHub(eh.Enable))
}

//Name returns the name of EdgeHub module
func (eh *EdgeHub) Name() string {
	return modules.EdgeHubModuleName
}

//Group returns EdgeHub group
func (eh *EdgeHub) Group() string {
	return modules.HubGroup
}

//Enable indicates whether this module is enabled
func (eh *EdgeHub) Enable() bool {
	return eh.enable
}

//Start sets context and starts the controller
func (eh *EdgeHub) Start() {
	eh.certManager = certificate.NewCertManager(config.Config.EdgeHub, config.Config.NodeName)
	eh.certManager.Start()

	HasTLSTunnelCerts <- true
	close(HasTLSTunnelCerts)

	go eh.ifRotationDone()

	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub stop")
			return
		default:
		}
		err := eh.initial()
		if err != nil {
			klog.Exitf("failed to init controller: %v", err)
			return
		}

		waitTime := time.Duration(config.Config.Heartbeat) * time.Second * 2

		//err = eh.chClient.Init()
		//if err != nil {
		//	klog.Errorf("connection failed: %v, will reconnect after %s", err, waitTime.String())
		//	time.Sleep(waitTime)
		//	continue
		//}

		klog.Infof("node relayconfig status", relayConfig.Config.GetStatus())
		klog.Infof("node isrelaynode", relayConfig.Config.GetIsRelayNode())

		if !relayConfig.Config.GetStatus() || (relayConfig.Config.GetStatus() && relayConfig.Config.GetIsRelayNode()) {
			klog.Infof("node open chClient")
			err = eh.chClient.Init()
			if err != nil {
				klog.Errorf("connection failed: %v, will reconnect after %s", err, waitTime.String())
				time.Sleep(waitTime)
				continue
			}
		} else {
			klog.Infof("non-relay no connection")
		}

		// execute hook func after connect
		eh.pubConnectInfo(true)
		go eh.switchRelayMode()
		if !relayConfig.Config.GetStatus() || (relayConfig.Config.GetStatus() && relayConfig.Config.GetIsRelayNode()) {
			klog.Infof("node open begin routeToEdge")
			go eh.routeToEdge()
		} else {
			klog.Infof("non-relay close routeToEdge")
		}
		// go eh.routeToEdge()
		go eh.routeToCloud()
		go eh.keepalive()

		// 如果开关处于开启阶段且本节点非中继节点，则越过正常连接阶段
		if relayConfig.Config.GetStatus() && !relayConfig.Config.GetIsRelayNode() {
			klog.Warningf("non-relay node wait for switching")

			<-relay.HubRelayChan.IsClose
			eh.pubConnectInfo(false)

			klog.Warningf("non-relay node mode switches")
			time.Sleep(waitTime)
		cleanChan:
			for {
				select {
				case <-relay.HubRelayChan.IsClose:
				default:
					break cleanChan
				}
			}

			continue // 跳出循环
		}

		// wait the stop signal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.UnInit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		klog.Warningf("connection is broken, will reconnect after %s", waitTime.String())
		time.Sleep(waitTime)

		// clean channel
	clean:
		for {
			select {
			case <-eh.reconnectChan:
			default:
				break clean
			}
		}
	}
}
