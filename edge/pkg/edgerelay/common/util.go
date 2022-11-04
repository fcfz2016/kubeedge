package common

import (
	"fmt"
	"k8s.io/klog/v2"
	"strings"
)

const (
	ResourceNode              = "node"
	ResourceTypeRelay         = "relayrcs"
	ResourceSep               = "/"
	ResourceNodeIDIndex       = 1
	ResourceResourceTypeIndex = 1

	RelayCloseOperation      = "closerelay"
	RelayOpenOperation       = "openrelay"
	RelayUpdateIDOperation   = "updateidrelay"
	RelayUpdateDataOperation = "updatedatarelay"

	GroupResource          = "relay"
	EdgeRelayModuleName    = "edgerelay"
	DefaultNameSpace       = "default"
	ResourceTypeRelayReply = "relayrcs"
	RelayReplyOperation    = "relayreply"

	ReplyOpenSuccess       = "opensuccess"
	ReplyCloseSuccess      = "closesuccess"
	ReplyFail              = "fail"
	ReplyChangeDataSuccess = "changedata"
	ReplyChangeIdSuccess   = "changeid"
)

func GetResourceType(resource string) string {
	sli := strings.Split(resource, ResourceSep)
	if len(sli) <= ResourceNodeIDIndex {
		klog.Errorf("no relayID")
		return ""
	}
	return sli[ResourceResourceTypeIndex]
}

func GetNodeID(resource string) string {
	sli := strings.Split(resource, ResourceSep)
	if len(sli) <= ResourceNodeIDIndex {
		klog.Errorf("no relayID")
		return ""
	}
	return sli[ResourceNodeIDIndex]
}

func IsFullResource(resource string) bool {
	sli := strings.Split(resource, ResourceSep)
	if len(sli) < 2 {
		return false
	}
	return true
}

func BuildResource(nodeID, namespace, resourceType, resourceID string) (resource string, err error) {
	if namespace == "" || resourceType == "" || nodeID == "" {
		err = fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
		return
	}

	resource = fmt.Sprintf("%s%s%s%s%s%s%s", ResourceNode, ResourceSep, nodeID, ResourceSep, namespace, ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", ResourceSep, resourceID)
	}
	return
}
