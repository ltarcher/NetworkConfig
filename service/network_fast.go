package service

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"sort"
	"strings"
)

// InterfaceFast 快速网卡信息结构
type InterfaceFast struct {
	Name        string `json:"name"`        // 网卡名称
	Status      string `json:"status"`      // 网卡状态(up/down)
	ProductName string `json:"productName"` // 网卡产品名称
}

// GetInterfacesFast 快速获取网卡列表
func (s *NetworkService) GetInterfacesFast() ([]InterfaceFast, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网卡列表失败: %v", err)
	}

	log.Printf("系统中共发现 %d 个网络接口", len(ifaces))

	var interfaces []InterfaceFast
	for _, iface := range ifaces {
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 根据操作系统跳过特定虚拟接口
		switch runtime.GOOS {
		case "windows":
			if strings.Contains(iface.Name, "Virtual") ||
				strings.Contains(strings.ToLower(iface.Name), "vethernet") ||
				strings.Contains(strings.ToLower(iface.Name), "wireguard") ||
				strings.Contains(strings.ToLower(iface.Name), "virtualbox") ||
				strings.Contains(strings.ToLower(iface.Name), "vmware") ||
				strings.Contains(strings.ToLower(iface.Name), "vpn") {
				continue
			}
		case "linux":
			if strings.HasPrefix(iface.Name, "docker") ||
				strings.HasPrefix(iface.Name, "veth") ||
				strings.HasPrefix(iface.Name, "br-") ||
				strings.HasPrefix(iface.Name, "virbr") ||
				strings.HasPrefix(iface.Name, "tun") {
				continue
			}
		}

		// 跳过MAC地址为空的无效网卡
		if iface.HardwareAddr == nil || len(iface.HardwareAddr) == 0 {
			log.Printf("跳过MAC地址为空的接口: %s", iface.Name)
			continue
		}

		// 跳过未启用的接口（除非在调试模式）
		// if !s.Debug && iface.Flags&net.FlagUp == 0 {
		//	log.Printf("跳过未启用的接口: %s", iface.Name)
		//	continue
		//}

		ifaceInfo := InterfaceFast{
			Name:   iface.Name,
			Status: getInterfaceStatusFast(iface.Flags),
		}
		// 获取硬件和驱动信息
		hardware, err := getHardwareInfo(iface.Name)
		if err != nil {
			log.Printf("获取接口 %s 硬件信息失败: %v", iface.Name, err)
		} else {
			log.Printf("接口 %s 硬件信息: %+v", iface.Name, hardware)
			ifaceInfo.ProductName = hardware.ProductName
			if hardware.ProductName == "" {
				log.Printf("警告: 接口 %s 的产品名称为空，忽略。", iface.Name)
				continue
			}
			if strings.Contains(ifaceInfo.ProductName, "KM-TEST") {
				log.Printf("警告: 接口 %s 的产品名称包含关键字 KM-TEST，忽略。", iface.Name)
				continue
			}
		}

		interfaces = append(interfaces, ifaceInfo)
	}

	if len(interfaces) == 0 {
		log.Println("警告: 没有找到可用的网络接口")
	}

	log.Printf("快速获取 %d 个网络接口的信息", len(interfaces))

	// 对网卡列表进行排序，WLAN网卡优先
	sort.Slice(interfaces, func(i, j int) bool {
		name1 := strings.ToLower(interfaces[i].Name)
		name2 := strings.ToLower(interfaces[j].Name)

		// 检查是否为WLAN网卡
		isWLAN1 := strings.Contains(name1, "wlan") ||
			strings.Contains(name1, "wi-fi") ||
			strings.Contains(name1, "wireless")
		isWLAN2 := strings.Contains(name2, "wlan") ||
			strings.Contains(name2, "wi-fi") ||
			strings.Contains(name2, "wireless")

		// WLAN网卡排在前面
		if isWLAN1 && !isWLAN2 {
			return true
		}
		if !isWLAN1 && isWLAN2 {
			return false
		}

		// 其他情况保持原顺序
		return i < j
	})

	return interfaces, nil
}

// getInterfaceStatusFast 快速获取接口状态
func getInterfaceStatusFast(flags net.Flags) string {
	if flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}
