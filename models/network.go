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
type Hardware struct {
	MACAddress    string `json:"mac_address"`
	Manufacturer  string `json:"manufacturer"`
	ProductName   string `json:"product_name"`
	AdapterType   string `json:"adapter_type"`
	PhysicalMedia string `json:"physical_media"`
	Speed         string `json:"speed"`
	BusType       string `json:"bus_type"`
	PNPDeviceID   string `json:"pnp_device_id"`
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
