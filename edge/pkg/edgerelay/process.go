package edgerelay

import (
	"bytes"
	"encoding/json"
	"fmt"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/common"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/viaduct/pkg/mux"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"time"
)

func (er *EdgeRelay) Load() {
	er.LoadRelayID()
	er.LoadData()
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
func (er *EdgeRelay) LoadRelayID() {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayID)
	if err != nil || len(*metas) == 0 {
		klog.Errorf("query relayID failed")
	}
	var result = *metas
	config.Config.SetRelayID(result[0])
}

func (er *EdgeRelay) SaveDate(data common.RelayData) {

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
		klog.Errorf("save relayId failed", err)
		return
	}
}
func (er *EdgeRelay) LoadData() {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayData)
	if err != nil || len(*metas) == 0 {
		klog.Errorf("query relayData failed")
	}
	var result = *metas
	data := &common.RelayData{}
	err = json.Unmarshal([]byte(result[0]), data)
	if err != nil {
		klog.Errorf("unmarshal relay data to json failed")
	}
	config.Config.SetData(*data)
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
		er.HandleMsgFromEdgeHub(&message)
	}

}

// HandleMsgFromEdgeHub
func (er *EdgeRelay) HandleMsgFromEdgeHub(msg *model.Message) {
	// 肯定是关于中继类型的信息，才会由EdgeHub发给Relay处理
	// 查看是否为中继模块下发节点信息
}

func (er *EdgeRelay) HandleMsgFromOtherEdge(container *mux.MessageContainer) {

}

func (er *EdgeRelay) server() {
	http.HandleFunc("/postMessage", er.receiveMessage)
	err := http.ListenAndServe("127.0.0.1:9090", nil)
	if err != nil {
		fmt.Println("net.Listen error :", err)
	}
}
func (er *EdgeRelay) receiveMessage(writer http.ResponseWriter, request *http.Request) {
	if request.Method == constants.POST {
		body, err := ioutil.ReadAll(request.Body)

		if err != nil {
			fmt.Println("Read failed:", err)
		}

		defer request.Body.Close()

		var container *mux.MessageContainer
		err = json.Unmarshal(body, container)

		if err != nil {
			fmt.Println("json format error:", err)
		}
		er.HandleMsgFromOtherEdge(container)

	}
}

// client
func (er *EdgeRelay) client(addr common.NodeAddress, container *mux.MessageContainer) {
	ip := addr.IP
	port := addr.Port

	var url string
	url = ip + strconv.FormatInt(port, 10) + "/postMessage"
	contentType := "application/json;charset=utf-8"

	b, err := json.Marshal(container)

	if err != nil {
		fmt.Println("json format error:", err)
	}

	body := bytes.NewBuffer(b)
	request, err := http.Post(url, contentType, body)
	if err != nil {
		fmt.Println("Post failed:", err)
	}
	defer request.Body.Close()

}

func (er *EdgeRelay) GetAddress(nodeID string) common.NodeAddress {
	var data = config.Config.GetData()
	return data.AddrData[nodeID]
}

// 从文件中读取所有的nodeID：addr键值对，返回一个map
func (er *EdgeRelay) GetAllAddress() map[string]common.NodeAddress {
	return config.Config.GetData().AddrData
}
