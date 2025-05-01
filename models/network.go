package models

// Interface 表示网卡信息
type Interface struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DHCPEnabled bool       `json:"dhcp_enabled"`
	IPv4Config  IPv4Config `json:"ipv4_config"`
	IPv6Config  IPv6Config `json:"ipv6_config"`
	Hardware    Hardware   `json:"hardware"`
	Driver      Driver     `json:"driver"`
}

// Hardware 表示网卡硬件信息
// AdapterType 定义网卡类型常量
const (
	AdapterTypeEthernet = "ethernet"
	AdapterTypeWireless = "wireless"
)

type Hardware struct {
	MACAddress    string `json:"mac_address"`    // MAC地址
	Manufacturer  string `json:"manufacturer"`   // 制造商
	ProductName   string `json:"product_name"`   // 产品名称
	AdapterType   string `json:"adapter_type"`   // 网卡类型: ethernet/wireless
	PhysicalMedia string `json:"physical_media"` // 物理介质
	Speed         string `json:"speed"`          // 连接速度
	BusType       string `json:"bus_type"`       // 总线类型
	PNPDeviceID   string `json:"pnp_device_id"`  // 即插即用设备ID
}

// Driver 表示网卡驱动信息
type Driver struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Provider      string `json:"provider"`
	DateInstalled string `json:"date_installed"`
	Status        string `json:"status"`
	Path          string `json:"path"`
}

// IPv4Config 表示IPv4配置信息
type IPv4Config struct {
	IP      string   `json:"ip"`
	Mask    string   `json:"mask"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
	DHCP    bool     `json:"dhcp"`
	DNSAuto bool     `json:"dnsAuto"`
}

// IPv6Config 表示IPv6配置信息
type IPv6Config struct {
	IP        string   `json:"ip"`
	PrefixLen int      `json:"prefix_len"`
	Gateway   string   `json:"gateway"`
	DNS       []string `json:"dns"`
}

// InterfaceConfig 表示网卡配置请求
type InterfaceConfig struct {
	IPv4Config *IPv4Config `json:"ipv4_config"`
	IPv6Config *IPv6Config `json:"ipv6_config"`
}

// ConnectivityResult 表示网络连通性探测结果
type ConnectivityResult struct {
	Target     string `json:"target"`      // 探测目标地址
	Success    bool   `json:"success"`     // 是否成功连接
	StatusCode int    `json:"status_code"` // HTTP状态码
	DurationMs int64  `json:"duration_ms"` // 响应时间(毫秒)
	Error      string `json:"error"`       // 错误信息(成功时为"")
}

// WiFiHotspot 表示可连接的WIFI热点信息
type WiFiHotspot struct {
	SSID         string `json:"ssid"`          // 热点名称
	BSSID        string `json:"bssid"`         // 热点MAC地址
	SignalLevel  int    `json:"signal_level"`  // 信号强度(百分比)
	Channel      int    `json:"channel"`       // 信道
	SecurityType string `json:"security_type"` // 加密类型(WPA2等)
	IsConnected  bool   `json:"is_connected"`  // 是否已连接
	Frequency    int    `json:"frequency"`     // 频率(MHz)
	RadioType    string `json:"radio_type"`    // 无线类型(802.11ac等)
}
