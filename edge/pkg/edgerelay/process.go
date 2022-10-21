package edgerelay

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	hubConfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/relay"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/common"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/relays/v1"
	"github.com/kubeedge/viaduct/pkg/mux"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (er *EdgeRelay) Load() {
	klog.Infof("start to load relay msg")
	er.LoadRelayStatus()
	er.LoadRelayID()
	er.LoadData()
}

func (er *EdgeRelay) Save(status bool, relayID string, relayData v1.RelayData) {
	klog.Infof("start to save relay msg")
	er.SaveRelayStatus(status)
	er.SaveRelayID(relayID)
	er.SaveDate(relayData)
}

func (er *EdgeRelay) UnMarshalMsg(msg *model.Message) (bool, string, v1.RelayData, error) {
	var relayrc v1.RelayrcSpec
	decodeBytes, err := Decode(msg)
	if err != nil {
		klog.Infof("RelayHandleServer:%v", err)
		return false, "", v1.RelayData{}, err
	}
	klog.Infof("edgerelay encode %v", decodeBytes)
	err = json.Unmarshal(decodeBytes, &relayrc)
	if err != nil {
		klog.Infof("RelayHandleServer:%v", err)
		return false, "", v1.RelayData{}, err
	}

	return relayrc.Open, relayrc.RelayID, relayrc.Data, nil
}

func (er *EdgeRelay) UnmarshalForForward(msg *model.Message) (*model.Message, error) {
	var rmsg model.Message
	decodeBytes, err := Decode(msg)
	if err != nil {
		klog.Infof("UnmarshalForForward:%v", err)
		return msg, err
	}
	klog.Infof("UnmarshalForForward encode %v", decodeBytes)
	err = json.Unmarshal(decodeBytes, &rmsg)
	if err != nil {
		klog.Infof("UnmarshalForForward:%v", err)
		return msg, err
	}
	return &rmsg, nil
}

func (er *EdgeRelay) LoadRelayStatus() {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayStatus)
	if err != nil || len(*metas) == 0 {
		klog.Errorf("query relayID failed")
		return
	}
	var result = *metas
	if result[0] == "1" {
		config.Config.SetStatus(true)
	} else {
		config.Config.SetStatus(false)
	}
	klog.Infof("load relayStatus", result[0])
}
func (er *EdgeRelay) SaveRelayStatus(relayStatus bool) {
	klog.Errorf("save relayStatus")
	config.Config.SetStatus(relayStatus)
	var stringStatus string
	if relayStatus {
		stringStatus = "1"
	} else {
		stringStatus = "0"
	}
	meta := &dao.Meta{
		Key:   constants.RelayStatus,
		Type:  constants.RelayType,
		Value: string(stringStatus),
	}
	err := dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("save relayStatus failed", err)
		return
	}
}

func (er *EdgeRelay) SaveRelayID(relayID string) {
	er.SetIsRelayNodeStatus()
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
	klog.Infof("save relayID, nodeid:%v, relayid:%v", config.Config.GetNodeID(), config.Config.GetRelayID())
}
func (er *EdgeRelay) LoadRelayID() {
	er.SetIsRelayNodeStatus()
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayID)
	if err != nil || len(*metas) == 0 {
		klog.Errorf("query relayID failed")
		return
	}
	var result = *metas
	config.Config.SetRelayID(result[0])
	klog.Infof("load relayID", result[0])
}

// 之后用于控制edgehub是否可以直连，在判断直连时，必须满足！isrelaynode&&relaystatus才确定为不直连
func (er *EdgeRelay) SetIsRelayNodeStatus() {
	if config.Config.GetRelayID() == config.Config.GetNodeID() {
		// 打开通道
		config.Config.SetIsRelayNode(true)
	} else {
		config.Config.SetIsRelayNode(false)
	}
}
func (er *EdgeRelay) ContinueEdgeHub() {
	relay.HubRelayChan.IsClose <- struct{}{}
}

func (er *EdgeRelay) SaveDate(data v1.RelayData) {
	klog.Errorf("save relayData")
	// 更新config和数据库
	config.Config.SetData(data)
	dataJson, err := json.Marshal(data)
	if err != nil {
		klog.Errorf("marshal relay data to json failed")
	}
	// 判断数据库中能不能查到，不能查到就insert，能查到就update
	meta := &dao.Meta{
		Key: constants.RelayData,
		// todo:确认type的含义
		Type:  constants.RelayType,
		Value: string(dataJson)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("save relayData failed", err)
		return
	}
}
func (er *EdgeRelay) LoadData() {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayData)
	if err != nil || len(*metas) == 0 {
		klog.Errorf("query relayData failed")
		return
	}
	var result = *metas
	data := v1.RelayData{}
	err = json.Unmarshal([]byte(result[0]), &data)
	if err != nil {
		klog.Errorf("unmarshal relay data to json failed")
	}
	config.Config.SetData(data)
	klog.Errorf("load relaydata", len(data.AddrData))
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
		if common.IsFullResource(message.GetResource()) {
			klog.Infof("edgerelay receive message from edgehub: %v", message.GetResource())
			klog.Infof("edgerelay receive message from edgehub: %v", message.GetContent())
			er.HandleMsgFromEdgeHub(&message)
		} else {
			klog.Errorf("edgerelay received message from edgehub is incomplete: %v", message.GetResource())
		}

	}

}

// HandleMsgFromEdgeHub
func (er *EdgeRelay) HandleMsgFromEdgeHub(msg *model.Message) {
	// 肯定是关于中继类型的信息，才会由EdgeHub发给Relay处理
	// 查看是否为中继模块下发节点信息
	if common.GetResourceType(msg.GetResource()) == common.ResourceTypeRelay {
		status, relayID, relayData, err := er.UnMarshalMsg(msg)
		if err != nil {
			return
		}
		switch msg.Router.Operation {
		case common.RelayCloseOperation:
			er.SaveRelayStatus(false)
			er.ContinueEdgeHub()
			break
		case common.RelayOpenOperation:
			er.Save(status, relayID, relayData)
			er.SetIsRelayNodeStatus()
			// er.ContinueEdgeHub()
			break
		case common.RelayUpdateDataOperation:
			er.SaveDate(relayData)
			break
		case common.RelayUpdateIDOperation:
			er.SaveRelayID(relayID)

			oldIsRelayNode := config.Config.GetIsRelayNode()
			er.SetIsRelayNodeStatus()
			if oldIsRelayNode != config.Config.GetIsRelayNode() {
				er.ContinueEdgeHub()
			}
			break
		}

		// 给其他节点下发中继信息
		klog.Infof("send relay_mark msg to non-relay node1")
		if config.Config.GetNodeID() == config.Config.GetRelayID() {
			klog.Infof("send relay_mark msg to non-relay node2")
			container := &mux.MessageContainer{
				Header:  map[string][]string{},
				Message: msg,
			}
			container.Header.Add("relay_mark", common.ResourceTypeRelay)
			nodeMap := er.GetAllAddress()
			for k, v := range nodeMap {
				// 给非本节点传递信息
				if k != config.Config.GetNodeID() {
					er.client(v, container)
				}
			}
			klog.Infof("send relay_mark msg finished, and feedback to cloud")
			er.replyToCloud()
		}

	} else {
		// 中继节点情况下：1、接收cloud传来的relay信息
		if config.Config.GetNodeID() == config.Config.GetRelayID() {
			msgFromConent, err := er.UnmarshalForForward(msg)
			if err != nil {
				klog.Errorf("edgerelay unmarshal msg from cloud to edge failed, %v", err)
			}

			container := &mux.MessageContainer{
				Header:  map[string][]string{},
				Message: msgFromConent,
			}
			nodeID := common.GetNodeID(msgFromConent.GetResource())

			trimMessage(msgFromConent)
			Encode(msgFromConent)

			nodeAddr := er.GetAddress(nodeID)
			er.client(nodeAddr, container)
			// 非中继节点情况下：2、将需要转发给cloud的信息，或keepalive信息传给中继节点处理
		} else {
			// else 封层container格式，添加自身nodeID和projectID，目标nodeID标为relayID
			container := &mux.MessageContainer{
				Header:  map[string][]string{},
				Message: msg,
			}
			container.Header.Add("node_id", config.Config.GetNodeID())
			container.Header.Add("project_id", hubConfig.Config.ProjectID)
			relayAddr := er.GetAddress(config.Config.GetRelayID())
			// 调用MsgToOtherEdge
			er.client(relayAddr, container)
		}
	}
}

func (er *EdgeRelay) HandleMsgFromOtherEdge(container *mux.MessageContainer) {
	relayMark := container.Header.Get("relay_mark")
	var msg *model.Message
	// 如果是节点标记信息
	if relayMark != "" {
		klog.Infof("non-relay node get relayMsg")
		msg = container.Message
		status, relayID, relayData, err := er.UnMarshalMsg(msg)
		if err != nil {
			klog.Errorf("relay_mark failed", err)
			return
		}
		switch msg.Router.Operation {
		case common.RelayCloseOperation:
			er.SaveRelayStatus(false)
			er.ContinueEdgeHub()
			break
		case common.RelayOpenOperation:
			er.Save(status, relayID, relayData)
			er.SetIsRelayNodeStatus()
			er.ContinueEdgeHub()
			break
		case common.RelayUpdateDataOperation:
			er.SaveDate(relayData)
			break
		case common.RelayUpdateIDOperation:
			er.SaveRelayID(relayID)

			oldIsRelayNode := config.Config.GetIsRelayNode()
			er.SetIsRelayNodeStatus()
			if oldIsRelayNode != config.Config.GetIsRelayNode() {
				er.ContinueEdgeHub()
			}
			break
		}

	} else {
		// else if(nodeID==relayID) 对信息进行封装，发送一个Operation为uploadrelay的Message，调用MsgToEdgeHub
		// 		else 对消息进行拆解，调用MsgToEdgeHub

		// 本节点是中继节点，负责把消息发给cloud端，ToCloud
		if config.Config.GetNodeID() == config.Config.GetRelayID() {
			msg = container.Message.Clone(container.Message)
			msg.SetResourceOperation(msg.GetResource(), constants.OpUploadRelayMessage)
			contentMsg, err := json.Marshal(*msg)

			if err != nil {
				fmt.Errorf("EdgeRelay SealMessage failed")
			}
			msg.Content = contentMsg
			er.MsgToEdgeHub(msg)
		} else {
			// 本节点非中继节点，ToOtherModule
			msg = container.Message
			er.MsgToOtherModule(msg)
		}
	}

}
func (er *EdgeRelay) MsgToEdgeHub(msg *model.Message) {
	// ch <- message
	_, err := beehiveContext.SendSync(modules.EdgeHubModuleName, *msg, 10*time.Second)
	if err != nil {
		klog.Errorf("send edgerelay msg to cloud failed,%v", err)
	}
}

func (er *EdgeRelay) MsgToOtherModule(msg *model.Message) {
	// 一个摆设cloudhubclient
	cloudHubClient, err := clients.GetClient()
	if err != nil {
		klog.Errorf("relay getclient error, discard: %v", err)
	}

	err = msghandler.ProcessHandler(*msg, cloudHubClient)
	if err != nil {
		klog.Errorf("relay failed to dispatch message, discard: %v", err)
	}
}

func (er *EdgeRelay) server() {
	klog.Infof("edgerelay server begin")
	http.HandleFunc("/postMessage", er.receiveMessage)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		fmt.Println("net.Listen error :", err)
	}
}
func (er *EdgeRelay) receiveMessage(writer http.ResponseWriter, request *http.Request) {
	klog.Infof("edgerelay receive from server")
	if request.Method == constants.POST {
		body, err := ioutil.ReadAll(request.Body)

		if err != nil {
			fmt.Println("Read failed:", err)
		}

		defer request.Body.Close()

		var container mux.MessageContainer
		err = json.Unmarshal(body, &container)

		if err != nil {
			fmt.Println("json format error:", err)
		}
		klog.Infof("edgerelay server msg to handle")
		er.HandleMsgFromOtherEdge(&container)

	}
}

// client
func (er *EdgeRelay) client(addr v1.NodeAddress, container *mux.MessageContainer) {
	ip := addr.IP
	port := addr.Port

	var url string
	url = "http://" + ip + ":" + strconv.FormatInt(port, 10) + "/postMessage"
	contentType := "application/json;charset=utf-8"

	b, err := json.Marshal(container)

	if err != nil {
		fmt.Println("json format error:", err)
	}

	body := bytes.NewBuffer(b)
	_, err = http.Post(url, contentType, body)

	if err != nil {
		klog.Errorf("Post failed:", err)
		return
		// todo: 多于某值的节点未收到的处理措施，以及失败重试
	}

}

func (er *EdgeRelay) GetAddress(nodeID string) v1.NodeAddress {
	var data = config.Config.GetData()
	return data.AddrData[nodeID]
}

// 从文件中读取所有的nodeID：addr键值对，返回一个map
func (er *EdgeRelay) GetAllAddress() map[string]v1.NodeAddress {
	return config.Config.GetData().AddrData
}

// 模拟cloud端下发时去掉目标node
func trimMessage(msg *model.Message) {
	resource := msg.GetResource()
	if strings.HasPrefix(resource, "node") {
		tokens := strings.Split(resource, "/")
		if len(tokens) < 3 {
			klog.Warningf("event resource %s starts with node but length less than 3", resource)
		} else {
			msg.SetResourceOperation(strings.Join(tokens[2:], "/"), msg.GetOperation())
		}
	}
}

func Decode(msg *model.Message) ([]byte, error) {
	v, ok := msg.GetContent().(string)
	if !ok {
		return nil, fmt.Errorf("assert failed")
	}
	decodeBytes, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("decode failed")
	}

	return decodeBytes, nil
}

func Encode(msg *model.Message) {
	encodeString := base64.StdEncoding.EncodeToString(msg.GetContent().([]byte))
	msg.Content = encodeString
}

func (er *EdgeRelay) replyToCloud() {
	msg := model.NewMessage("")

	resource, err := common.BuildResource(config.Config.GetNodeID(), common.DefaultNameSpace, common.ResourceTypeRelayReply, "")
	if err != nil {
		klog.Warningf("replytocloud built message resource failed with error: %s", err)
		return
	}
	msg = msg.BuildRouter(common.EdgeRelayModuleName, common.GroupResource, resource, common.RelayReplyOperation)
	msg.Content = "OK"

	er.MsgToEdgeHub(msg)
}
