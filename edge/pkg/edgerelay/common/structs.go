package common

type NodeAddress struct {
	// Required
	IP string `json:"ip,omitempty"`
	// Required
	Port int64 `json:"port,omitempty"`
}
type RelayData struct {
	// todo:other attribute
	AddrData map[string]NodeAddress `json:"addrdata,omitempty"`
}
