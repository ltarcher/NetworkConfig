package models

import (
	"encoding/json"
	"testing"
)

func TestInterfaceJSON(t *testing.T) {
	// 测试用例：无线网卡（已连接热点）
	wirelessIface := Interface{
		Name:          "Wi-Fi",
		Description:   "Intel(R) Wireless-AC 9560",
		Status:        "up",
		ConnectedSSID: "TestWiFi",
		IPv4Config: IPv4Config{
			IP:      "192.168.1.100",
			Mask:    "255.255.255.0",
			Gateway: "192.168.1.1",
			DNS:     []string{"8.8.8.8", "8.8.4.4"},
		},
		IPv6Config: IPv6Config{
			IP:        "2001:db8::1",
			PrefixLen: 64,
			Gateway:   "2001:db8::1",
			DNS:       []string{"2001:4860:4860::8888"},
		},
		Hardware: Hardware{
			MACAddress:    "00:11:22:33:44:55",
			Manufacturer:  "Intel Corporation",
			ProductName:   "Intel(R) Wireless-AC 9560",
			AdapterType:   AdapterTypeWireless,
			PhysicalMedia: "Native 802.11",
			Speed:         "866.7 Mbps",
			BusType:       "PCI",
			PNPDeviceID:   "PCI\\VEN_8086&DEV_A370",
		},
		Driver: Driver{
			Name:          "Intel(R) Wireless-AC 9560",
			Version:       "22.10.0.7",
			Provider:      "Intel",
			DateInstalled: "2024-01-01",
			Status:        "OK",
			Path:          "C:\\Windows\\System32\\DriverStore\\FileRepository\\netwtw10.inf_amd64_abc123\\netwtw10.inf",
		},
	}

	// 测试用例：有线网卡
	ethernetIface := Interface{
		Name:        "Ethernet",
		Description: "Intel(R) Ethernet Connection I219-V",
		Status:      "up",
		IPv4Config: IPv4Config{
			IP:      "192.168.1.100",
			Mask:    "255.255.255.0",
			Gateway: "192.168.1.1",
			DNS:     []string{"8.8.8.8", "8.8.4.4"},
		},
		Hardware: Hardware{
			MACAddress:    "00:11:22:33:44:66",
			Manufacturer:  "Intel Corporation",
			ProductName:   "Intel(R) Ethernet Connection I219-V",
			AdapterType:   AdapterTypeEthernet,
			PhysicalMedia: "Ethernet",
			Speed:         "1000 Mbps",
			BusType:       "PCI",
			PNPDeviceID:   "PCI\\VEN_8086&DEV_15B8",
		},
		Driver: Driver{
			Name:          "Intel(R) Ethernet Connection I219-V",
			Version:       "12.18.9.23",
			Provider:      "Intel",
			DateInstalled: "2024-01-01",
			Status:        "OK",
			Path:          "C:\\Windows\\System32\\DriverStore\\FileRepository\\e1d68x64.inf_amd64_abc123\\e1d68x64.inf",
		},
	}

	// 测试无线网卡JSON序列化/反序列化
	t.Run("Wireless Interface", func(t *testing.T) {
		data, err := json.Marshal(wirelessIface)
		if err != nil {
			t.Errorf("无线网卡JSON序列化失败: %v", err)
		}

		var decoded Interface
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Errorf("无线网卡JSON反序列化失败: %v", err)
		}

		// 验证字段值
		if wirelessIface.Name != decoded.Name {
			t.Errorf("Name字段不匹配: 期望 %s, 得到 %s", wirelessIface.Name, decoded.Name)
		}
		if wirelessIface.ConnectedSSID != decoded.ConnectedSSID {
			t.Errorf("ConnectedSSID字段不匹配: 期望 %s, 得到 %s", wirelessIface.ConnectedSSID, decoded.ConnectedSSID)
		}
		if wirelessIface.Hardware.AdapterType != decoded.Hardware.AdapterType {
			t.Errorf("AdapterType字段不匹配: 期望 %s, 得到 %s", wirelessIface.Hardware.AdapterType, decoded.Hardware.AdapterType)
		}
	})

	// 测试有线网卡JSON序列化/反序列化
	t.Run("Ethernet Interface", func(t *testing.T) {
		data, err := json.Marshal(ethernetIface)
		if err != nil {
			t.Errorf("有线网卡JSON序列化失败: %v", err)
		}

		var decoded Interface
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Errorf("有线网卡JSON反序列化失败: %v", err)
		}

		// 验证字段值
		if ethernetIface.Name != decoded.Name {
			t.Errorf("Name字段不匹配: 期望 %s, 得到 %s", ethernetIface.Name, decoded.Name)
		}
		if decoded.ConnectedSSID != "" {
			t.Errorf("有线网卡不应该有ConnectedSSID字段值，但得到了: %s", decoded.ConnectedSSID)
		}
		if ethernetIface.Hardware.AdapterType != decoded.Hardware.AdapterType {
			t.Errorf("AdapterType字段不匹配: 期望 %s, 得到 %s", ethernetIface.Hardware.AdapterType, decoded.Hardware.AdapterType)
		}
	})
}
