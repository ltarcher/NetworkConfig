package service

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
)

// InterfaceFast 快速网卡信息结构
type InterfaceFast struct {
	Name   string `json:"name"`   // 网卡名称
	Status string `json:"status"` // 网卡状态(up/down)
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
			   strings.Contains(strings.ToLower(iface.Name), "wireguard") {
				continue
			}
		case "linux":
			if strings.HasPrefix(iface.Name, "docker") || 
			   strings.HasPrefix(iface.Name, "veth") {
				continue
			}
		}

		interfaces = append(interfaces, InterfaceFast{
			Name:   iface.Name,
			Status: getInterfaceStatusFast(iface.Flags),
		})
	}

	if len(interfaces) == 0 {
		log.Println("警告: 没有找到可用的网络接口")
	}

	log.Printf("快速获取 %d 个网络接口的信息", len(interfaces))
	return interfaces, nil
}

// getInterfaceStatusFast 快速获取接口状态
func getInterfaceStatusFast(flags net.Flags) string {
	if flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}