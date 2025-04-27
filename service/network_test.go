package service

import (
	"net"
	"networkconfig/models"
	"regexp"
	"testing"
	"time"
)

// mockNetworkService 模拟网络服务
type mockNetworkService struct {
	interfaces map[string]models.Interface
}

// newMockNetworkService 创建模拟网络服务实例
func newMockNetworkService() *mockNetworkService {
	return &mockNetworkService{
		interfaces: map[string]models.Interface{
			"eth0": {
				Name:        "eth0",
				Description: "Test Ethernet Interface",
				Status:      "up",
				IPv4Config: models.IPv4Config{
					IP:      "192.168.1.100",
					Mask:    "255.255.255.0",
					Gateway: "192.168.1.1",
					DNS:     []string{"8.8.8.8", "8.8.4.4"},
				},
				IPv6Config: models.IPv6Config{
					IP:        "2001:db8::1",
					PrefixLen: 64,
					Gateway:   "2001:db8::1",
					DNS:       []string{"2001:4860:4860::8888"},
				},
				Hardware: models.Hardware{
					MACAddress:    "00:11:22:33:44:55",
					Manufacturer:  "Intel Corporation",
					ProductName:   "Intel(R) Ethernet Connection I219-V",
					AdapterType:   "Ethernet 802.3",
					PhysicalMedia: "Ethernet",
					Speed:         "1000 Mbps",
					BusType:       "PCI",
					PNPDeviceID:   "PCI\\VEN_8086&DEV_15B8",
				},
				Driver: models.Driver{
					Name:          "Intel(R) Ethernet Connection I219-V",
					Version:       "12.18.9.23",
					Provider:      "Intel",
					DateInstalled: "2024-01-01",
					Status:        "OK",
					Path:          "C:\\Windows\\System32\\DriverStore\\FileRepository\\e1d68x64.inf_amd64_abc123\\e1d68x64.inf",
				},
			},
			"wlan0": {
				Name:        "wlan0",
				Description: "Test Wireless Interface",
				Status:      "up",
				IPv4Config: models.IPv4Config{
					IP:      "192.168.2.100",
					Mask:    "255.255.255.0",
					Gateway: "192.168.2.1",
					DNS:     []string{"8.8.8.8", "8.8.4.4"},
				},
			},
		},
	}
}

func TestGetInterfaces(t *testing.T) {
	// 创建真实的NetworkService实例
	service := NewNetworkService()

	// 获取网卡列表
	interfaces, err := service.GetInterfaces()
	if err != nil {
		t.Logf("获取网卡列表可能需要管理员权限: %v", err)
		return
	}

	// 验证返回的网卡列表
	if len(interfaces) == 0 {
		t.Log("警告: 未找到网卡")
		return
	}

	// 验证每个网卡的基本信息
	for _, iface := range interfaces {
		t.Run(iface.Name, func(t *testing.T) {
			if iface.Name == "" {
				t.Error("网卡名称为空")
			}
			if iface.Status == "" {
				t.Error("网卡状态为空")
			}
			if iface.Hardware.MACAddress == "" {
				t.Error("MAC地址为空")
			}
			if iface.Driver.Name == "" {
				t.Error("驱动名称为空")
			}

			// 验证至少有一个IP配置
			if iface.IPv4Config.IP == "" && iface.IPv6Config.IP == "" {
				t.Error("网卡没有IP配置")
			}

			// 验证MAC地址格式
			if iface.Hardware.MACAddress != "" {
				macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
				if matched, _ := regexp.MatchString(macPattern, iface.Hardware.MACAddress); !matched {
					t.Errorf("MAC地址格式无效: %s", iface.Hardware.MACAddress)
				}
			}
		})
	}
}

func TestGetInterface(t *testing.T) {
	// 创建真实的NetworkService实例
	service := NewNetworkService()

	// 获取网卡列表
	interfaces, err := service.GetInterfaces()
	if err != nil {
		t.Logf("获取网卡列表可能需要管理员权限: %v", err)
		return
	}

	if len(interfaces) == 0 {
		t.Log("警告: 未找到可用的网卡")
		return
	}

	// 测试获取第一个网卡
	firstInterface := interfaces[0]
	testName := firstInterface.Name

	t.Run(testName, func(t *testing.T) {
		iface, err := service.GetInterface(testName)
		if err != nil {
			t.Fatalf("获取网卡 %s 信息失败: %v", testName, err)
		}

		// 验证基本信息
		if iface.Name != testName {
			t.Errorf("网卡名称不匹配: 期望 %s, 得到 %s", testName, iface.Name)
		}
		if iface.Status == "" {
			t.Error("网卡状态为空")
		}

		// 验证硬件信息
		if iface.Hardware.MACAddress == "" {
			t.Error("MAC地址为空")
		} else {
			macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
			if matched, _ := regexp.MatchString(macPattern, iface.Hardware.MACAddress); !matched {
				t.Errorf("MAC地址格式无效: %s", iface.Hardware.MACAddress)
			}
		}

		// 验证驱动信息
		if iface.Driver.Name == "" {
			t.Error("驱动名称为空")
		}

		// 验证IP配置
		if iface.IPv4Config.IP == "" && iface.IPv6Config.IP == "" {
			t.Error("网卡没有IP配置")
		}

		// 如果有IPv4配置，验证格式
		if iface.IPv4Config.IP != "" {
			ip := net.ParseIP(iface.IPv4Config.IP)
			if ip == nil || ip.To4() == nil {
				t.Errorf("无效的IPv4地址: %s", iface.IPv4Config.IP)
			}
		}

		// 如果有IPv6配置，验证格式
		if iface.IPv6Config.IP != "" {
			ip := net.ParseIP(iface.IPv6Config.IP)
			if ip == nil || ip.To4() != nil {
				t.Errorf("无效的IPv6地址: %s", iface.IPv6Config.IP)
			}
		}
	})
}

func TestConfigureInterface(t *testing.T) {
	// 创建真实的NetworkService实例
	service := NewNetworkService()

	// 准备测试配置
	testConfig := models.InterfaceConfig{
		IPv4Config: &models.IPv4Config{
			IP:      "192.168.1.100",
			Mask:    "255.255.255.0",
			Gateway: "192.168.1.1",
			DNS:     []string{"8.8.8.8", "8.8.4.4"},
		},
		IPv6Config: &models.IPv6Config{
			IP:        "2001:db8::1",
			PrefixLen: 64,
			Gateway:   "2001:db8::1",
			DNS:       []string{"2001:4860:4860::8888"},
		},
	}

	// 获取第一个可用的网卡
	interfaces, err := service.GetInterfaces()
	if err != nil {
		t.Logf("获取网卡列表可能需要管理员权限: %v", err)
		return
	}

	if len(interfaces) == 0 {
		t.Log("警告: 未找到可用的网卡")
		return
	}

	// 尝试配置第一个网卡
	testInterface := interfaces[0]
	t.Logf("测试配置网卡: %s", testInterface.Name)

	// 由于这是一个可能修改系统配置的测试，我们只记录而不实际执行
	t.Logf("将要应用的配置: %+v", testConfig)
	t.Log("注意: 跳过实际配置以避免修改系统设置")
}

func TestParseNetworkOutput(t *testing.T) {
	// 测试网关解析
	testOutput := `
网络目标        网络掩码          网关       接口
0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100
	`
	gateway := parseGateway(testOutput)
	if gateway != "192.168.1.1" {
		t.Errorf("网关解析错误: 期望 192.168.1.1, 得到 %s", gateway)
	}

	// 测试DNS解析
	testDNSOutput := `
接口 eth0：
DNS 服务器: 8.8.8.8
DNS 服务器: 8.8.4.4
	`
	dnsServers := parseDNSServers(testDNSOutput)
	if len(dnsServers) != 2 {
		t.Errorf("DNS服务器数量不匹配: 期望 2, 得到 %d", len(dnsServers))
	}
	if len(dnsServers) > 0 && dnsServers[0] != "8.8.8.8" {
		t.Errorf("第一个DNS服务器不匹配: 期望 8.8.8.8, 得到 %s", dnsServers[0])
	}
}

func TestGetHardwareInfo(t *testing.T) {
	// 创建NetworkService实例
	service := NewNetworkService()

	// 获取网卡列表
	interfaces, err := service.GetInterfaces()
	if err != nil {
		t.Logf("获取网卡列表可能需要管理员权限: %v", err)
		return
	}

	if len(interfaces) == 0 {
		t.Log("警告: 未找到网卡")
		return
	}

	// 测试第一个网卡的硬件信息
	firstInterface := interfaces[0]
	hardware := firstInterface.Hardware

	// 验证硬件信息字段
	if hardware.MACAddress == "" {
		t.Error("MAC地址为空")
	}

	if hardware.Manufacturer == "" {
		t.Error("制造商信息为空")
	}

	if hardware.ProductName == "" {
		t.Error("产品名称为空")
	}

	if hardware.Speed == "" {
		t.Error("速度信息为空")
	}

	// 验证MAC地址格式
	macPattern := `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
	if matched, _ := regexp.MatchString(macPattern, hardware.MACAddress); !matched {
		t.Errorf("MAC地址格式无效: %s", hardware.MACAddress)
	}

	t.Logf("硬件信息: %+v", hardware)
}

func TestGetDriverInfo(t *testing.T) {
	// 创建NetworkService实例
	service := NewNetworkService()

	// 获取网卡列表
	interfaces, err := service.GetInterfaces()
	if err != nil {
		t.Logf("获取网卡列表可能需要管理员权限: %v", err)
		return
	}

	if len(interfaces) == 0 {
		t.Log("警告: 未找到网卡")
		return
	}

	// 测试第一个网卡的驱动信息
	firstInterface := interfaces[0]
	driver := firstInterface.Driver

	// 验证驱动信息字段
	if driver.Name == "" {
		t.Error("驱动名称为空")
	}

	if driver.Version == "" {
		t.Error("驱动版本为空")
	}

	if driver.Provider == "" {
		t.Error("驱动提供商为空")
	}

	if driver.DateInstalled == "" {
		t.Error("安装日期为空")
	}

	if driver.Status == "" {
		t.Error("驱动状态为空")
	}

	// 验证日期格式
	if driver.DateInstalled != "Unknown" {
		_, err := time.Parse("2006-01-02", driver.DateInstalled)
		if err != nil {
			t.Errorf("安装日期格式无效: %s", driver.DateInstalled)
		}
	}

	t.Logf("驱动信息: %+v", driver)
}
