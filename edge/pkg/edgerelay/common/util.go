package common

import (
	"k8s.io/klog/v2"
	"strings"
)

const (
	ResourceTypeRelay         = "relayres"
	ResourceSep               = "/"
	ResourceNodeIDIndex       = 1
	ResourceOperationIndex    = 1
	ResourceResourceTypeIndex = 3

	RelayCloseOperation      = "closerelay"
	RelayOpenOperation       = "openrelay"
	RelayUpdateIDOperation   = "updateidrelay"
	RelayUpdateDataOperation = "updatedatarelay"

	GroupResource = "relay"
)

func GetResourceType(resource string) string {
	sli := strings.Split(resource, ResourceSep)
	if len(sli) <= ResourceNodeIDIndex {
		klog.Errorf("no relayID")
	}
	return sli[ResourceResourceTypeIndex]
}

func GetNodeID(resource string) string {
	sli := strings.Split(resource, ResourceSep)
	if len(sli) <= ResourceNodeIDIndex {
		klog.Errorf("no relayID")
	}
	return sli[ResourceNodeIDIndex]
}
