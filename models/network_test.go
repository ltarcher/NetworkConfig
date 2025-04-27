package models

import (
	"encoding/json"
	"testing"
)

func TestInterfaceJSON(t *testing.T) {
	// 创建测试数据
	iface := Interface{
		Name:        "Test Interface",
		Description: "Test Description",
		Status:      "up",
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
			ProductName:   "Intel(R) Ethernet Connection I219-V",
			AdapterType:   "Ethernet 802.3",
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

	// 测试JSON序列化
	data, err := json.Marshal(iface)
	if err != nil {
		t.Errorf("JSON序列化失败: %v", err)
	}

	// 测试JSON反序列化
	var decodedIface Interface
	err = json.Unmarshal(data, &decodedIface)
	if err != nil {
		t.Errorf("JSON反序列化失败: %v", err)
	}

	// 验证字段值
	if iface.Name != decodedIface.Name {
		t.Errorf("Name字段不匹配: 期望 %s, 得到 %s", iface.Name, decodedIface.Name)
	}
	if iface.Description != decodedIface.Description {
		t.Errorf("Description字段不匹配: 期望 %s, 得到 %s", iface.Description, decodedIface.Description)
	}
	if iface.Status != decodedIface.Status {
		t.Errorf("Status字段不匹配: 期望 %s, 得到 %s", iface.Status, decodedIface.Status)
	}

	// 验证IPv4配置
	if iface.IPv4Config.IP != decodedIface.IPv4Config.IP {
		t.Errorf("IPv4 IP字段不匹配: 期望 %s, 得到 %s", iface.IPv4Config.IP, decodedIface.IPv4Config.IP)
	}
	if iface.IPv4Config.Mask != decodedIface.IPv4Config.Mask {
		t.Errorf("IPv4 Mask字段不匹配: 期望 %s, 得到 %s", iface.IPv4Config.Mask, decodedIface.IPv4Config.Mask)
	}
	if iface.IPv4Config.Gateway != decodedIface.IPv4Config.Gateway {
		t.Errorf("IPv4 Gateway字段不匹配: 期望 %s, 得到 %s", iface.IPv4Config.Gateway, decodedIface.IPv4Config.Gateway)
	}

	// 验证IPv6配置
	if iface.IPv6Config.IP != decodedIface.IPv6Config.IP {
		t.Errorf("IPv6 IP字段不匹配: 期望 %s, 得到 %s", iface.IPv6Config.IP, decodedIface.IPv6Config.IP)
	}
	if iface.IPv6Config.PrefixLen != decodedIface.IPv6Config.PrefixLen {
		t.Errorf("IPv6 PrefixLen字段不匹配: 期望 %d, 得到 %d", iface.IPv6Config.PrefixLen, decodedIface.IPv6Config.PrefixLen)
	}
	if iface.IPv6Config.Gateway != decodedIface.IPv6Config.Gateway {
		t.Errorf("IPv6 Gateway字段不匹配: 期望 %s, 得到 %s", iface.IPv6Config.Gateway, decodedIface.IPv6Config.Gateway)
	}
}