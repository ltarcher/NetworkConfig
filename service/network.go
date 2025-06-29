package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/url"
	"networkconfig/models"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// bytesToHexString 将字节数组转换为十六进制字符串
func bytesToHexString(b []byte) string {
	var buf bytes.Buffer
	for _, b := range b {
		buf.WriteString(fmt.Sprintf("%02X", b))
	}
	return buf.String()
}

// 定义服务错误
var (
	ErrInterfaceNotFound = errors.New("interface not found")
)

// NetworkService 处理网络配置相关的操作
type NetworkService struct {
	Debug          bool            // 调试模式开关，true时获取网卡列表不进行过滤
	hotspotMonitor *HotspotMonitor // 热点监控服务
}

// NewNetworkService 创建新的NetworkService实例
// debug参数控制调试模式，true时获取网卡列表不进行过滤
func NewNetworkService(debug bool) *NetworkService {
	service := &NetworkService{
		Debug: debug,
	}

	// 创建热点监控服务
	service.hotspotMonitor = NewHotspotMonitor(service, debug)

	return service
}

// StartHotspotMonitor 启动热点监控服务
func (s *NetworkService) StartHotspotMonitor() {
	if s.hotspotMonitor != nil {
		s.hotspotMonitor.Start()
	}
}

// StopHotspotMonitor 停止热点监控服务
func (s *NetworkService) StopHotspotMonitor() {
	if s.hotspotMonitor != nil {
		s.hotspotMonitor.Stop()
	}
}

// GetInterfaces 获取所有网卡信息
func (s *NetworkService) GetInterfaces() ([]models.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网卡列表失败: %v", err)
	}

	log.Printf("系统中共发现 %d 个网络接口", len(ifaces))

	var interfaces []models.Interface
	for _, iface := range ifaces {
		log.Printf("正在处理接口: %s (MTU: %d, Flags: %v)", iface.Name, iface.MTU, iface.Flags)

		// 调试模式下跳过所有过滤
		if !s.Debug {
			// 跳过回环接口
			if iface.Flags&net.FlagLoopback != 0 {
				log.Printf("跳过回环接口: %s", iface.Name)
				continue
			}

			// 跳过Virtual虚拟接口
			if strings.Contains(iface.Name, "Virtual") {
				log.Printf("跳过Virtual虚拟接口: %s", iface.Name)
				continue
			}

			// 跳过WireGuard接口
			if strings.Contains(strings.ToLower(iface.Name), "wireguard") {
				log.Printf("跳过WireGuard接口: %s", iface.Name)
				continue
			}

			// 跳过未启用的接口
			if iface.Flags&net.FlagUp == 0 {
				log.Printf("未启用的接口: %s", iface.Name)
				// continue
			}
		}

		ifaceInfo, err := s.GetInterface(iface.Name)
		if err != nil {
			log.Printf("获取接口 %s 信息失败: %v", iface.Name, err)

			// 创建基本接口信息
			//basicInfo := models.Interface{
			//	Name:        iface.Name,
			//	Description: iface.Name,
			//	Status:      getInterfaceStatus(iface.Flags),
			//	Hardware: models.Hardware{
			//		MACAddress: iface.HardwareAddr.String(),
			//	},
			//	Driver: models.Driver{
			//		Name: iface.Name,
			//	},
			//}

			// 获取硬件信息失败的，可能是虚拟网卡，跳过
			// interfaces = append(interfaces, basicInfo)
			continue
		}

		// 检查MAC地址和产品名称是否为空
		if ifaceInfo.Hardware.MACAddress == "" {
			log.Printf("跳过MAC地址为空的接口: %s", iface.Name)
			continue
		}

		if ifaceInfo.Hardware.ProductName == "" {
			log.Printf("跳过产品名称为空的接口: %s", iface.Name)
			continue
		}

		if strings.Compare(ifaceInfo.Hardware.ProductName, "KM-TEST") == 0 {
			log.Printf("跳过产品名称包含关键字 KM-TEST 的接口: %s", iface.Name)
			continue
		}

		interfaces = append(interfaces, ifaceInfo)
	}

	if len(interfaces) == 0 {
		log.Println("警告: 没有找到可用的网络接口")
	}

	log.Printf("成功获取 %d 个网络接口的信息", len(interfaces))
	return interfaces, nil
}

// GetInterface 获取指定网卡的详细信息
func (s *NetworkService) GetInterface(name string) (models.Interface, error) {
	log.Printf("开始获取接口 %s 的信息", name)

	iface, err := net.InterfaceByName(name)
	if err != nil {
		log.Printf("获取网卡 %s 信息失败: %v", name, err)
		return models.Interface{}, fmt.Errorf("获取网卡信息失败: %v", err)
	}

	log.Printf("接口 %s 基本信息: MTU=%d, Flags=%v, HardwareAddr=%s",
		name, iface.MTU, iface.Flags, iface.HardwareAddr)

	addrs, err := iface.Addrs()
	if err != nil {
		log.Printf("获取网卡 %s 地址失败: %v", name, err)
		return models.Interface{}, fmt.Errorf("获取网卡地址失败: %v", err)
	}

	log.Printf("接口 %s 有 %d 个地址", name, len(addrs))

	// 检查DHCP状态
	dhcpEnabled := false
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "config", "name="+name)
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "DHCP enabled") {
				dhcpEnabled = strings.Contains(line, "Yes")
				break
			}
		}
	}

	ifaceInfo := models.Interface{
		Name:        iface.Name,
		Description: getInterfaceDescription(name),
		Status:      getInterfaceStatus(iface.Flags),
		DHCPEnabled: dhcpEnabled,
	}

	// 获取硬件和驱动信息
	hardware, err := getHardwareInfo(name)
	if err != nil {
		log.Printf("获取接口 %s 硬件信息失败: %v", name, err)
		ifaceInfo.Hardware = models.Hardware{
			MACAddress: iface.HardwareAddr.String(),
		}
		return ifaceInfo, fmt.Errorf("获取硬件信息失败: %v", err)
	} else {
		ifaceInfo.Hardware = hardware
		log.Printf("接口 %s 硬件信息: %+v", name, hardware)

		// 如果是无线网卡，获取当前连接的SSID
		if hardware.AdapterType == models.AdapterTypeWireless {
			ssid, err := getConnectedSSID(name)
			if err != nil {
				log.Printf("获取接口 %s 的SSID失败: %v", name, err)
			} else if ssid != "" {
				ifaceInfo.ConnectedSSID = ssid
				log.Printf("接口 %s 当前连接的热点: %s", name, ssid)
			}
		}
	}

	//driver, err := getDriverInfo(name)
	//if err != nil {
	//	log.Printf("获取接口 %s 驱动信息失败: %v", name, err)
	//	ifaceInfo.Driver = models.Driver{
	//		Name: iface.Name,
	//	}
	//} else {
	//	ifaceInfo.Driver = driver
	//	log.Printf("接口 %s 驱动信息: %+v", name, driver)
	//}

	// 获取IPv4和IPv6配置
	for i, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			log.Printf("接口 %s 地址 %d 不是有效的IPNet", name, i)
			continue
		}

		if ipNet.IP.To4() != nil {
			// IPv4
			gateway := getDefaultGateway(name)
			dns := getDNSServers(name)
			log.Printf("接口 %s IPv4地址: IP=%s, Mask=%s, Gateway=%s, DNS=%v",
				name, ipNet.IP, net.IP(ipNet.Mask), gateway, dns)

			ifaceInfo.IPv4Config = models.IPv4Config{
				IP:      ipNet.IP.String(),
				Mask:    net.IP(ipNet.Mask).String(),
				Gateway: gateway,
				DNS:     dns,
			}
		} else {
			// IPv6
			prefixLen, _ := ipNet.Mask.Size()
			gateway := getIPv6Gateway(name)
			dns := getIPv6DNSServers(name)
			log.Printf("接口 %s IPv6地址: IP=%s, PrefixLen=%d, Gateway=%s, DNS=%v",
				name, ipNet.IP, prefixLen, gateway, dns)

			ifaceInfo.IPv6Config = models.IPv6Config{
				IP:        ipNet.IP.String(),
				PrefixLen: prefixLen,
				Gateway:   gateway,
				DNS:       dns,
			}
		}
	}

	log.Printf("成功获取接口 %s 的完整信息", name)
	return ifaceInfo, nil
}

// getHardwareInfo 获取网卡硬件信息
func getHardwareInfo(name string) (models.Hardware, error) {
	// 首先尝试使用PowerShell获取信息
	hw, err := getHardwareInfoViaPowerShell(name)
	if err == nil {
		return hw, nil
	}

	log.Printf("通过PowerShell获取接口 %s 硬件信息失败: %v，尝试备用方案", name, err)

	// 检查是否是无线网卡
	if isWirelessInterface(name) {
		// 尝试通过netsh获取无线网卡信息
		hw, err := getWirelessInfoViaNetsh(name)
		if err == nil {
			log.Printf("成功通过netsh获取接口 %s 的无线网卡信息", name)
			return hw, nil
		}
		log.Printf("通过netsh获取接口 %s 无线网卡信息失败: %v", name, err)
	}

	// 如果都失败，返回最少信息
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return models.Hardware{}, fmt.Errorf("无法获取网卡基本信息: %v", err)
	}

	return models.Hardware{
		MACAddress: iface.HardwareAddr.String(),
	}, nil
}

// getHardwareInfoViaPowerShell 通过PowerShell获取硬件信息
func getHardwareInfoViaPowerShell(name string) (models.Hardware, error) {
	// 使用PowerShell命令获取网卡硬件信息，设置UTF-8编码
	psCmd := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$PSDefaultParameterValues['*:Encoding'] = 'utf8'
		Get-WmiObject Win32_NetworkAdapter | Where-Object { $_.NetConnectionID -eq '%s' -or $_.Name -eq '%s' } | 
		Select-Object MACAddress,Manufacturer,ProductName,AdapterType,NetConnectionID,Speed,PNPDeviceID | 
		ConvertTo-Json -Depth 1
	`, name, name)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	log.Printf("执行PowerShell命令获取网卡 %s 的硬件信息", name)
	output, err := cmd.Output()
	if err != nil {
		// 获取错误详情
		if exitErr, ok := err.(*exec.ExitError); ok {
			return models.Hardware{}, fmt.Errorf("执行PowerShell命令失败: %v, stderr: %s", err, string(exitErr.Stderr))
		}
		return models.Hardware{}, fmt.Errorf("执行PowerShell命令失败: %v", err)
	}

	if len(output) == 0 {
		log.Printf("未找到网卡 %s 的硬件信息", name)
		return models.Hardware{}, fmt.Errorf("未找到网卡硬件信息: %s", name)
	}

	// 尝试转换编码
	decodedOutput, err := DecodeToUTF8(output)
	if err != nil {
		log.Printf("转换编码失败: %v", err)
		return models.Hardware{}, fmt.Errorf("转换编码失败: %v", err)
	}

	log.Printf("网卡 %s 的原始硬件信息: %s", name, string(decodedOutput))

	// 解析JSON输出
	var result struct {
		MACAddress   string `json:"MACAddress"`
		Manufacturer string `json:"Manufacturer"`
		ProductName  string `json:"ProductName"`
		AdapterType  string `json:"AdapterType"`
		Speed        uint64 `json:"Speed"`
		PNPDeviceID  string `json:"PNPDeviceID"`
	}

	if err := json.Unmarshal(decodedOutput, &result); err != nil {
		log.Printf("解析硬件信息JSON失败: %v", err)
		return models.Hardware{}, fmt.Errorf("解析硬件信息失败: %v", err)
	}

	log.Printf("成功解析网卡 %s 的硬件信息: %+v", name, result)

	// 获取物理媒体类型
	mediaCmd := exec.Command("powershell", "-Command",
		fmt.Sprintf(`Get-WmiObject Win32_NetworkAdapter | Where-Object { $_.NetConnectionID -eq '%s' } | Select-Object PhysicalAdapter | ConvertTo-Json`, name))

	mediaOutput, err := mediaCmd.Output()
	if err == nil {
		var mediaResult struct {
			PhysicalAdapter bool `json:"PhysicalAdapter"`
		}
		if err := json.Unmarshal(mediaOutput, &mediaResult); err == nil {
			if mediaResult.PhysicalAdapter {
				// 获取总线类型
				// 获取总线类型，设置UTF-8编码
				busCmd := fmt.Sprintf(`
					[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
					$PSDefaultParameterValues['*:Encoding'] = 'utf8'
					Get-WmiObject Win32_NetworkAdapter | 
						Where-Object { $_.NetConnectionID -eq '%s' } | 
						Select-Object Caption | 
						ConvertTo-Json -Depth 1
				`, name)

				cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", busCmd)
				cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

				if busOutput, err := cmd.Output(); err == nil {
					// 转换编码
					decodedBusOutput, err := DecodeToUTF8(busOutput)
					if err == nil {
						var busResult struct {
							Caption string `json:"Caption"`
						}
						if err := json.Unmarshal(decodedBusOutput, &busResult); err == nil {
							log.Printf("网卡 %s 的总线信息: %s", name, busResult.Caption)
							// 从Caption中提取总线类型
							if strings.Contains(busResult.Caption, "PCI") {
								result.AdapterType = "PCI"
							} else if strings.Contains(busResult.Caption, "USB") {
								result.AdapterType = "USB"
							}
						} else {
							log.Printf("解析总线信息JSON失败: %v", err)
						}
					} else {
						log.Printf("转换总线信息编码失败: %v", err)
					}
				} else {
					log.Printf("获取总线信息失败: %v", err)
				}
			}
		}
	}

	// 转换速度为可读格式
	speedStr := "Unknown"
	if result.Speed > 0 {
		speed := float64(result.Speed) / 1000000 // 转换为Mbps
		speedStr = fmt.Sprintf("%.0f Mbps", speed)
	}

	// 确定网卡类型
	adapterType := models.AdapterTypeEthernet // 默认为有线
	if strings.Contains(strings.ToLower(result.ProductName), "wireless") ||
		strings.Contains(strings.ToLower(result.ProductName), "wi-fi") ||
		strings.Contains(strings.ToLower(result.ProductName), "wlan") {
		adapterType = models.AdapterTypeWireless
	}

	return models.Hardware{
		MACAddress:    result.MACAddress,
		Manufacturer:  result.Manufacturer,
		ProductName:   result.ProductName,
		AdapterType:   adapterType,
		PhysicalMedia: "Ethernet", // 默认值，可以根据实际情况修改
		Speed:         speedStr,
		BusType:       result.AdapterType,
		PNPDeviceID:   result.PNPDeviceID,
	}, nil
}

// isWirelessInterface 判断是否是无线网卡
func isWirelessInterface(name string) bool {
	// 根据常见无线网卡命名规则判断
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "wi-fi") ||
		strings.Contains(lowerName, "wireless") ||
		strings.Contains(lowerName, "wlan")
}

// getWirelessInfoViaNetsh 通过netsh获取无线网卡信息
func getWirelessInfoViaNetsh(interfaceName string) (models.Hardware, error) {
	log.Printf("尝试通过netsh获取接口 %s 的无线网卡信息", interfaceName)

	// 获取所有无线网卡接口信息
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("netsh命令执行失败: %v, 输出: %s", err, string(output))
		return models.Hardware{}, fmt.Errorf("netsh命令执行失败: %v", err)
	}

	// 将输出转换为字符串
	outputStr := string(output)
	log.Printf("netsh原始输出:\n%s", outputStr)

	// 按接口分割输出
	interfaces := strings.Split(outputStr, "\n\n")
	var targetOutput string
	found := false

	// 遍历每个接口块，查找指定的网卡
	for _, iface := range interfaces {
		// 从接口块中提取网卡名称
		lines := strings.Split(iface, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "名称") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) > 1 {
					name := strings.TrimSpace(parts[1])
					log.Printf("检查接口: %q 是否匹配目标: %q", name, interfaceName)
					if name == interfaceName {
						targetOutput = iface
						found = true
						break
					}
				}
			}
		}
		if found {
			break
		}
	}

	if !found {
		log.Printf("在可用的无线网卡列表中未找到接口 %s", interfaceName)
		return models.Hardware{}, fmt.Errorf("指定的网卡 %s 不是可用的无线网卡", interfaceName)
	}

	log.Printf("找到目标网卡 %s 的信息块:\n%s", interfaceName, targetOutput)
	return parseWirelessNetshOutput(targetOutput), nil
}

// parseWirelessNetshOutput 解析netsh命令输出
func parseWirelessNetshOutput(output string) models.Hardware {
	hw := models.Hardware{
		AdapterType: models.AdapterTypeWireless,
	}

	log.Printf("开始解析无线网卡信息块...")
	lines := strings.Split(output, "\n")
	var rxRate, txRate string
	var manufacturer string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Description", "描述":
			hw.ProductName = value
			// 尝试从描述中提取制造商信息
			if strings.Contains(value, "Intel") {
				manufacturer = "Intel Corporation"
			} else if strings.Contains(value, "Realtek") {
				manufacturer = "Realtek Semiconductor Corp."
			} else if strings.Contains(value, "Broadcom") {
				manufacturer = "Broadcom Inc."
			} else if strings.Contains(value, "MediaTek") {
				manufacturer = "MediaTek Inc."
			}
			log.Printf("解析到产品名称: %s", value)

		case "Name", "名称":
			if hw.ProductName == "" {
				hw.ProductName = value
				log.Printf("使用网卡名称作为产品名称: %s", value)
			}

		case "Physical address", "物理地址":
			hw.MACAddress = value
			log.Printf("解析到MAC地址: %s", value)

		case "Media type", "媒体类型", "Connection type", "连接类型":
			hw.PhysicalMedia = value
			log.Printf("解析到媒体类型: %s", value)

		case "State", "状态":
			// 记录状态但不存储，可用于调试
			log.Printf("网卡状态: %s", value)

		case "SSID", "SSID 名称":
			// 记录当前连接的SSID，可用于调试
			log.Printf("当前连接的SSID: %s", value)

		case "Receive rate (Mbps)", "接收速率 (Mbps)":
			rxRate = value
			log.Printf("解析到接收速率: %s Mbps", value)

		case "Transmit rate (Mbps)", "传输速率 (Mbps)":
			txRate = value
			log.Printf("解析到传输速率: %s Mbps", value)

		case "Signal", "信号":
			// 记录信号强度，可用于调试
			log.Printf("当前信号强度: %s", value)

		case "Band", "频段":
			// 记录频段信息，可用于调试
			log.Printf("工作频段: %s", value)

		case "Radio type", "无线电类型":
			// 可以用来确定是802.11n/ac等
			log.Printf("无线电类型: %s", value)
			if hw.PhysicalMedia == "" {
				hw.PhysicalMedia = fmt.Sprintf("802.11 %s", value)
			}
		}
	}

	// 设置制造商信息
	if manufacturer != "" {
		hw.Manufacturer = manufacturer
		log.Printf("设置制造商: %s", manufacturer)
	}

	// 组合速率信息
	if rxRate != "" || txRate != "" {
		var speedParts []string
		if rxRate != "" {
			speedParts = append(speedParts, fmt.Sprintf("Rx: %s Mbps", rxRate))
		}
		if txRate != "" {
			speedParts = append(speedParts, fmt.Sprintf("Tx: %s Mbps", txRate))
		}
		hw.Speed = strings.Join(speedParts, ", ")
		log.Printf("设置最终速率: %s", hw.Speed)
	}

	// 设置总线类型为PCI（大多数无线网卡都是PCI设备）
	hw.BusType = "PCI"

	// 验证必要字段
	if hw.ProductName == "" {
		log.Printf("警告: 未能解析到产品名称")
	}
	if hw.MACAddress == "" {
		log.Printf("警告: 未能解析到MAC地址")
	}
	if hw.PhysicalMedia == "" {
		hw.PhysicalMedia = "802.11 Wireless"
		log.Printf("设置默认媒体类型: %s", hw.PhysicalMedia)
	}

	log.Printf("无线网卡信息解析完成: %+v", hw)
	return hw
}

// getDriverInfo 获取网卡驱动信息
func getDriverInfo(name string) (models.Driver, error) {
	log.Printf("开始获取网卡 %s 的驱动信息", name)

	// 使用PowerShell命令获取网卡驱动信息，设置UTF-8编码
	psCmd := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$PSDefaultParameterValues['*:Encoding'] = 'utf8'
		$ErrorActionPreference = 'Stop'
		
		# 首先获取网络适配器的PNPDeviceID
		$adapter = Get-WmiObject Win32_NetworkAdapter | Where-Object { $_.NetConnectionID -eq '%s' -or $_.Name -eq '%s' }
		if ($adapter) {
			# 使用PNPDeviceID查找对应的驱动程序
			Get-WmiObject Win32_PnPSignedDriver | 
				Where-Object { $_.DeviceID -eq $adapter.PNPDeviceID } |
				Select-Object DriverVersion,DriverProvider,DriverDate,DeviceName,InfName |
				ConvertTo-Json -Depth 1
		} else {
			Write-Error "找不到指定的网络适配器"
		}
	`, name, name)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psCmd)
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"POWERSHELL_TELEMETRY_OPTOUT=1")

	output, err := cmd.Output()
	if err != nil {
		// 获取错误详情
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			log.Printf("获取网卡 %s 驱动信息时出错: %v\nstderr: %s", name, err, stderr)
			if strings.Contains(stderr, "找不到指定的网络适配器") {
				return models.Driver{}, fmt.Errorf("找不到网卡: %s", name)
			}
			return models.Driver{}, fmt.Errorf("获取驱动信息失败: %v", err)
		}
		log.Printf("执行PowerShell命令失败: %v", err)
		return models.Driver{}, fmt.Errorf("执行PowerShell命令失败: %v", err)
	}

	if len(output) == 0 {
		log.Printf("未找到网卡 %s 的驱动信息", name)
		return models.Driver{}, fmt.Errorf("未找到网卡 %s 的驱动信息", name)
	}

	// 尝试转换编码
	decodedOutput, err := DecodeToUTF8(output)
	if err != nil {
		log.Printf("转换驱动信息编码失败: %v", err)
		return models.Driver{}, fmt.Errorf("转换编码失败: %v", err)
	}

	log.Printf("网卡 %s 的原始驱动信息: %s", name, string(decodedOutput))

	// 解析JSON输出
	var result struct {
		DriverVersion  string `json:"DriverVersion"`
		DriverProvider string `json:"DriverProvider"`
		DriverDate     string `json:"DriverDate"`
		DeviceName     string `json:"DeviceName"`
		InfName        string `json:"InfName"`
	}

	if err := json.Unmarshal(decodedOutput, &result); err != nil {
		log.Printf("解析驱动信息JSON失败: %v", err)
		return models.Driver{}, fmt.Errorf("解析驱动信息失败: %v", err)
	}

	log.Printf("成功解析网卡 %s 的驱动信息: %+v", name, result)

	// 格式化安装日期
	dateInstalled := "Unknown"
	if result.DriverDate != "" {
		if date, err := time.Parse("20060102150405.999999-070", result.DriverDate); err == nil {
			dateInstalled = date.Format("2006-01-02")
		}
	}

	return models.Driver{
		Name:          result.DeviceName,
		Version:       result.DriverVersion,
		Provider:      result.DriverProvider,
		DateInstalled: dateInstalled,
		Status:        "OK", // 默认值，可以根据实际情况修改
		Path:          result.InfName,
	}, nil
}

// ConfigureInterface 配置网卡
func (s *NetworkService) ConfigureInterface(name string, config models.InterfaceConfig) error {
	// 添加原始请求日志
	raw, _ := json.Marshal(config)
	log.Printf("原始请求体JSON: %s", string(raw))

	// 添加详细调试日志
	log.Printf("接收到接口 %s 的完整配置请求:", name)
	if config.IPv4Config != nil {
		log.Printf("IPv4配置: IP=%s, Mask=%s, Gateway=%s, DNS=%v, DHCP=%v, DNSAuto=%v",
			config.IPv4Config.IP,
			config.IPv4Config.Mask,
			config.IPv4Config.Gateway,
			config.IPv4Config.DNS,
			config.IPv4Config.DHCP,
			config.IPv4Config.DNSAuto)
	}
	if config.IPv6Config != nil {
		log.Printf("IPv6配置: IP=%s, PrefixLen=%d, Gateway=%s, DNS=%v",
			config.IPv6Config.IP,
			config.IPv6Config.PrefixLen,
			config.IPv6Config.Gateway,
			config.IPv6Config.DNS)
	}

	if config.IPv4Config != nil {
		log.Printf("IPv4配置详情: IP=%s, Mask=%s, Gateway=%s, DNS=%v, DHCP=%v, DNSAuto=%v",
			config.IPv4Config.IP,
			config.IPv4Config.Mask,
			config.IPv4Config.Gateway,
			config.IPv4Config.DNS,
			config.IPv4Config.DHCP,
			config.IPv4Config.DNSAuto)

		if err := s.configureIPv4(name, *config.IPv4Config); err != nil {
			return fmt.Errorf("配置IPv4失败: %v", err)
		}
	}

	if config.IPv6Config != nil {
		log.Printf("IPv6配置详情: IP=%s, PrefixLen=%d, Gateway=%s, DNS=%v",
			config.IPv6Config.IP,
			config.IPv6Config.PrefixLen,
			config.IPv6Config.Gateway,
			config.IPv6Config.DNS)

		if err := s.configureIPv6(name, *config.IPv6Config); err != nil {
			return fmt.Errorf("配置IPv6失败: %v", err)
		}
	}

	return nil
}

// configureIPv4 配置IPv4地址
func (s *NetworkService) configureIPv4(name string, config models.IPv4Config) error {
	if config.DHCP {
		log.Printf("开始为接口 %s 配置DHCP自动获取IP", name)

		// 检查当前是否已经是DHCP状态
		currentDHCP, err := isDHCPEnabled(name)
		if err != nil {
			log.Printf("检查接口 %s 的DHCP状态失败: %v", name, err)
			return fmt.Errorf("检查DHCP状态失败: %v", err)
		}

		if !currentDHCP {
			// 当前不是DHCP状态，需要设置
			log.Printf("为接口 %s 设置DHCP自动获取IP", name)

			cmd := exec.Command("netsh",
				"interface",
				"ipv4",
				"set",
				"address",
				fmt.Sprintf("name=%s", name),
				"source=dhcp")

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("设置DHCP失败: %v, 输出: %s", err, string(output))
				return fmt.Errorf("设置DHCP失败: %v, 输出: %s", err, string(output))
			}
			log.Printf("成功设置DHCP自动获取IP")
		} else {
			log.Printf("接口 %s 已经是DHCP状态，跳过设置", name)
		}

		// 设置DNS
		if config.DNSAuto {
			cmdStr := fmt.Sprintf("netsh interface ipv4 set dnsservers name=\"%s\" source=dhcp", name)
			log.Printf("执行命令: %s", cmdStr)

			cmd := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers",
				fmt.Sprintf("name=%s", name),
				"source=dhcp")

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("设置DNS自动获取失败: %v, 输出: %s", err, string(output))
				return fmt.Errorf("设置DNS自动获取失败: %v, 输出: %s", err, string(output))
			}
			log.Printf("成功设置DNS自动获取")
		} else if len(config.DNS) > 0 {
			var cmdStr string
			log.Printf("开始设置指定DNS服务器: %v", config.DNS)
			for i, dns := range config.DNS {
				var cmd *exec.Cmd
				if i == 0 {
					cmdStr = fmt.Sprintf("netsh interface ipv4 set dns name=\"%s\" static %s",
						name, dns)
					cmd = exec.Command("netsh", "interface", "ipv4", "set", "dns",
						fmt.Sprintf("name=%s", name),
						"static",
						dns)
				} else {
					cmdStr = fmt.Sprintf("netsh interface ipv4 add dns name=\"%s\" %s index=%d",
						name, dns, i+1)
					cmd = exec.Command("netsh", "interface", "ipv4", "add", "dns",
						fmt.Sprintf("name=%s", name),
						dns,
						fmt.Sprintf("index=%d", i+1))
				}
				log.Printf("执行命令: %s", cmdStr)

				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("设置指定DNS服务器失败: %v, 输出: %s", err, string(output))
					return fmt.Errorf("设置指定DNS服务器失败: %v, 输出: %s", err, string(output))
				}
			}
			log.Printf("成功设置所有指定DNS服务器")
		}
	} else {
		log.Printf("开始配置接口 %s 的静态IPv4设置: IP=%s, Mask=%s, Gateway=%s, DNS=%v",
			name, config.IP, config.Mask, config.Gateway, config.DNS)

		// 设置静态IP地址和子网掩码
		cmdStr := fmt.Sprintf("netsh interface ipv4 set address name=\"%s\" static %s %s %s",
			name, config.IP, config.Mask, config.Gateway)
		log.Printf("执行命令: %s", cmdStr)

		// 验证接口是否存在
		if _, err := net.InterfaceByName(name); err != nil {
			return fmt.Errorf("接口 %s 不存在: %v", name, err)
		}

		// 构造netsh命令参数
		args := []string{
			"interface",
			"ipv4",
			"set",
			"address",
			fmt.Sprintf("name=%s", name), // 直接传递接口名称
			"static",
			config.IP,
			config.Mask,
		}
		if config.Gateway != "" {
			args = append(args, config.Gateway)
		}

		// 记录完整命令
		log.Printf("执行命令: netsh %v", args)

		cmd := exec.Command("netsh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("命令执行失败: %v\n完整命令: netsh %v\n输出: %s",
				err, args, string(output))
			return fmt.Errorf("设置静态IPv4地址失败: %v, 输出: %s", err, string(output))
		}
		log.Printf("成功设置静态IPv4地址")

		// 设置静态DNS服务器
		if len(config.DNS) > 0 {
			log.Printf("开始设置静态DNS服务器: %v", config.DNS)
			for i, dns := range config.DNS {
				var cmd *exec.Cmd
				if i == 0 {
					cmdStr = fmt.Sprintf("netsh interface ipv4 set dns name=\"%s\" static %s",
						name, dns)
					cmd = exec.Command("netsh", "interface", "ipv4", "set", "dns",
						fmt.Sprintf("name=%s", name),
						"static",
						dns)
				} else {
					cmdStr = fmt.Sprintf("netsh interface ipv4 add dns name=\"%s\" %s index=%d",
						name, dns, i+1)
					cmd = exec.Command("netsh", "interface", "ipv4", "add", "dns",
						fmt.Sprintf("name=%s", name),
						dns,
						fmt.Sprintf("index=%d", i+1))
				}
				log.Printf("执行命令: %s", cmdStr)

				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("设置静态DNS服务器失败: %v, 输出: %s", err, string(output))
					return fmt.Errorf("设置静态DNS服务器失败: %v, 输出: %s", err, string(output))
				}
			}
			log.Printf("成功设置所有静态DNS服务器")
		}
	}

	log.Printf("接口 %s 的IPv4配置完成", name)
	return nil
}

// configureIPv6 配置IPv6地址
func (s *NetworkService) configureIPv6(name string, config models.IPv6Config) error {
	// 设置IPv6地址
	cmd := exec.Command("netsh", "interface", "ipv6", "set", "address",
		fmt.Sprintf("interface=%s", name),
		fmt.Sprintf("address=%s", config.IP),
		"store=persistent")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置IPv6地址失败: %v", err)
	}

	// 设置IPv6网关
	if config.Gateway != "" {
		cmd = exec.Command("netsh", "interface", "ipv6", "add", "route",
			"::/0",
			fmt.Sprintf("interface=%s", name),
			config.Gateway)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("设置IPv6网关失败: %v", err)
		}
	}

	// 设置IPv6 DNS服务器
	if len(config.DNS) > 0 {
		for i, dns := range config.DNS {
			cmd := exec.Command("netsh", "interface", "ipv6", "set", "dns",
				fmt.Sprintf("name=%s", name),
				"static",
				dns)
			if i > 0 {
				cmd = exec.Command("netsh", "interface", "ipv6", "add", "dns",
					fmt.Sprintf("name=%s", name),
					dns,
					fmt.Sprintf("index=%d", i+1))
			}
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("设置IPv6 DNS服务器失败: %v", err)
			}
		}
	}

	return nil
}

// 辅助函数

func getInterfaceDescription(name string) string {
	cmd := exec.Command("netsh", "interface", "show", "interface", name)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	// 解析输出获取描述信息
	return strings.TrimSpace(string(output))
}

func getInterfaceStatus(flags net.Flags) string {
	if flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}

func getDefaultGateway(name string) string {
	// 方法1: 使用netsh命令
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "route", name)
	output, err := cmd.Output()
	if err == nil {
		gateway := parseGateway(string(output))
		if gateway != "" {
			log.Printf("通过netsh获取到接口 %s 的网关: %s", name, gateway)
			return gateway
		}
	} else {
		log.Printf("netsh获取网关失败: %v", err)
	}

	// 方法2: 使用route print命令
	cmd = exec.Command("route", "print", "-4")
	outputBytes, err := cmd.Output()
	if err == nil {
		output := string(outputBytes)
		gateway := parseGateway(output)
		if gateway != "" {
			log.Printf("通过route print获取到接口 %s 的网关: %s", name, gateway)
			return gateway
		}
	} else {
		log.Printf("route print获取网关失败: %v", err)
	}

	// 方法3: 使用ipconfig命令
	cmd = exec.Command("ipconfig")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if strings.Contains(line, name) {
				// 查找后续的默认网关行
				for j := i + 1; j < len(lines); j++ {
					if strings.Contains(lines[j], "默认网关") ||
						strings.Contains(lines[j], "Default Gateway") {
						parts := strings.Split(lines[j], ":")
						if len(parts) > 1 {
							gateway := strings.TrimSpace(parts[1])
							if gateway != "" {
								log.Printf("通过ipconfig获取到接口 %s 的网关: %s", name, gateway)
								return gateway
							}
						}
					}
				}
			}
		}
	} else {
		log.Printf("ipconfig获取网关失败: %v", err)
	}

	log.Printf("无法获取接口 %s 的网关", name)
	return ""
}

func getIPv6Gateway(name string) string {
	cmd := exec.Command("netsh", "interface", "ipv6", "show", "route", name)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	// 解析输出获取IPv6默认网关
	return parseGateway(string(output))
}

func getDNSServers(name string) []string {
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "dnsservers", fmt.Sprintf("name=\"%s\"", name))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("获取接口 %s 的DNS服务器失败: %v", name, err)
		return []string{"unavailable"}
	}

	servers := []string{}
	lines := strings.Split(string(output), "\n")
	inDnsSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查是否进入DNS服务器部分
		if strings.Contains(line, "静态配置的 DNS 服务器") ||
			strings.Contains(line, "Statically Configured DNS Servers") {
			inDnsSection = true
			continue
		}

		// 只在DNS服务器部分处理
		if inDnsSection {
			// 跳过说明行和空行
			if line == "" ||
				strings.Contains(line, "用哪个前缀注册") ||
				strings.Contains(line, "Register with which suffix") {
				continue
			}

			// 提取IP地址
			if ip := net.ParseIP(line); ip != nil {
				servers = append(servers, ip.String())
			} else {
				// 处理可能的多行格式
				parts := strings.Fields(line)
				for _, part := range parts {
					if ip := net.ParseIP(part); ip != nil {
						servers = append(servers, ip.String())
					}
				}
			}
		}
	}

	// 如果没有找到DNS服务器，尝试备用方法
	if len(servers) == 0 {
		servers = getDNSServersAlternative(name)
	}

	if len(servers) == 0 {
		return []string{"none"}
	}
	return servers
}

// 备用DNS获取方法
func getDNSServersAlternative(name string) []string {
	// 方法1: 使用ipconfig /all
	cmd := exec.Command("ipconfig", "/all")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		inInterfaceSection := false
		servers := []string{}

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// 检查是否进入目标接口部分
			if strings.Contains(line, name) {
				inInterfaceSection = true
				continue
			}

			if inInterfaceSection {
				// 检查DNS服务器行
				if strings.Contains(line, "DNS Servers") || strings.Contains(line, "DNS 服务器") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						ip := strings.TrimSpace(parts[1])
						if net.ParseIP(ip) != nil {
							servers = append(servers, ip)
						}
					}
				}

				// 检查是否离开接口部分
				if strings.Contains(line, "----------") {
					break
				}
			}
		}

		if len(servers) > 0 {
			return servers
		}
	}

	// 方法2: 使用Get-DnsClientServerAddress PowerShell命令
	psCmd := fmt.Sprintf(`
        [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
        $PSDefaultParameterValues['*:Encoding'] = 'utf8'
        (Get-DnsClientServerAddress -InterfaceAlias "%s" -AddressFamily IPv4).ServerAddresses
    `, name)

	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	output, err = cmd.Output()
	if err == nil {
		// 解析输出，每行一个IP地址
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		servers := []string{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if ip := net.ParseIP(line); ip != nil {
				servers = append(servers, ip.String())
			}
		}
		return servers
	}

	return []string{}
}

func getIPv6DNSServers(name string) []string {
	cmd := exec.Command("netsh", "interface", "ipv6", "show", "dnsservers", fmt.Sprintf("name=\"%s\"", name))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("获取接口 %s 的IPv6 DNS服务器失败: %v", name, err)
		return []string{"unavailable"}
	}

	servers := []string{}
	lines := strings.Split(string(output), "\n")
	inDnsSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查是否进入DNS服务器部分
		if strings.Contains(line, "静态配置的 DNS 服务器") ||
			strings.Contains(line, "Statically Configured DNS Servers") {
			inDnsSection = true
			continue
		}

		// 只在DNS服务器部分处理
		if inDnsSection {
			// 跳过说明行和空行
			if line == "" ||
				strings.Contains(line, "用哪个前缀注册") ||
				strings.Contains(line, "Register with which suffix") {
				continue
			}

			// 提取IP地址
			if ip := net.ParseIP(line); ip != nil {
				servers = append(servers, ip.String())
			} else {
				// 处理可能的多行格式
				parts := strings.Fields(line)
				for _, part := range parts {
					if ip := net.ParseIP(part); ip != nil {
						servers = append(servers, ip.String())
					}
				}
			}
		}
	}

	// 如果没有找到DNS服务器，尝试备用方法
	if len(servers) == 0 {
		servers = getIPv6DNSServersAlternative(name)
	}

	if len(servers) == 0 {
		return []string{"none"}
	}
	return servers
}

// 备用IPv6 DNS获取方法
func getIPv6DNSServersAlternative(name string) []string {
	// 方法1: 使用ipconfig /all
	cmd := exec.Command("ipconfig", "/all")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		inInterfaceSection := false
		servers := []string{}

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// 检查是否进入目标接口部分
			if strings.Contains(line, name) {
				inInterfaceSection = true
				continue
			}

			if inInterfaceSection {
				// 检查IPv6 DNS服务器行
				if strings.Contains(line, "DNS Servers") || strings.Contains(line, "DNS 服务器") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						ip := strings.TrimSpace(parts[1])
						if net.ParseIP(ip) != nil && strings.Contains(ip, ":") { // 确保是IPv6地址
							servers = append(servers, ip)
						}
					}
				}

				// 检查是否离开接口部分
				if strings.Contains(line, "----------") {
					break
				}
			}
		}

		if len(servers) > 0 {
			return servers
		}
	}

	// 方法2: 使用Get-DnsClientServerAddress PowerShell命令
	psCmd := fmt.Sprintf(`
        [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
        $PSDefaultParameterValues['*:Encoding'] = 'utf8'
        (Get-DnsClientServerAddress -InterfaceAlias "%s" -AddressFamily IPv6).ServerAddresses
    `, name)

	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	output, err = cmd.Output()
	if err == nil {
		// 解析输出，每行一个IP地址
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		servers := []string{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if ip := net.ParseIP(line); ip != nil && ip.To4() == nil { // 确保是IPv6地址
				servers = append(servers, ip.String())
			}
		}
		return servers
	}

	return []string{}
}

func parseGateway(output string) string {
	lines := strings.Split(output, "\n")

	// 尝试匹配不同格式的网关输出
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 格式1: 0.0.0.0/0 <metric> <interface> <gateway>
		if strings.HasPrefix(line, "0.0.0.0/0") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				return fields[3]
			}
		}

		// 格式2: 0.0.0.0 <mask> <gateway> <interface> <metric>
		if strings.HasPrefix(line, "0.0.0.0") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[2]
			}
		}

		// 格式3: 默认网关: <gateway>
		if strings.HasPrefix(line, "默认网关:") || strings.HasPrefix(line, "Default Gateway:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	// 如果上述方法都失败，尝试使用route print命令
	cmd := exec.Command("route", "print", "0.0.0.0")
	outputBytes, err := cmd.Output()
	if err == nil {
		output := string(outputBytes)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 3 && fields[0] == "0.0.0.0" {
					return fields[2]
				}
			}
		}
	}

	return ""
}

// isDHCPEnabled 检查指定网络接口是否启用了DHCP
func isDHCPEnabled(name string) (bool, error) {
	// 使用netsh命令检查接口配置
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "config", "name="+name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("执行netsh命令失败: %v, 输出: %s", err, string(output))
	}

	// 解析输出查找DHCP状态
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "DHCP enabled") {
			// 检查是否包含"Yes"表示启用
			return strings.Contains(line, "Yes"), nil
		}
	}

	return false, fmt.Errorf("无法从输出中确定DHCP状态: %s", string(output))
}

// CheckConnectivity 检查网络连通性
// WiFiHotspot 表示WiFi热点信息
type WiFiHotspot struct {
	SSID           string `json:"ssid"`
	SignalStrength int    `json:"signal_strength"` // 信号强度百分比
	Security       string `json:"security"`        // 加密类型
	BSSID          string `json:"bssid"`           // MAC地址
	Channel        int    `json:"channel"`         // 信道
}

func (s *NetworkService) GetWiFiHotspots(interfaceName string) ([]WiFiHotspot, error) {
	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)

	// 根据操作系统执行不同命令
	switch runtime.GOOS {
	case "windows":
		return s.scanWiFiWindows(interfaceName)
	case "linux":
		return s.scanWiFiLinux(interfaceName)
	default:
		return hotspots, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (s *NetworkService) scanWiFiWindows(interfaceName string) ([]WiFiHotspot, error) {
	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)

	log.Printf("开始扫描接口 %s 的WiFi热点...", interfaceName)

	// 构造命令
	args := []string{
		"wlan", "show", "networks",
		"mode=bssid",
		fmt.Sprintf("interface=%s", interfaceName),
	}
	cmd := exec.Command("netsh", args...)
	log.Printf("执行命令: netsh %v", args)

	// 执行命令并捕获输出
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("WiFi扫描命令执行失败: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("命令错误输出: %s", string(exitErr.Stderr))
		}
		return hotspots, fmt.Errorf("扫描WiFi失败: %v", err)
	}

	// 记录原始输出用于调试
	rawOutput := string(out)
	log.Printf("WiFi扫描原始输出(前100字符): %q...", safeSubstring(rawOutput, 100))
	if len(rawOutput) > 1000 {
		log.Printf("完整输出已记录到调试日志")
	}

	// 解析输出
	hotspots, err = parseNetshOutput(rawOutput)
	if err != nil {
		log.Printf("解析WiFi扫描输出失败: %v", err)
		return []WiFiHotspot{}, fmt.Errorf("解析WiFi扫描结果失败: %v", err)
	}

	log.Printf("成功扫描到 %d 个WiFi热点", len(hotspots))
	return hotspots, nil
}

// safeSubstring 安全截取字符串，避免索引越界
func safeSubstring(s string, length int) string {
	if length <= 0 {
		return ""
	}
	if len(s) <= length {
		return s
	}
	return s[:length]
}

func (s *NetworkService) scanWiFiLinux(interfaceName string) ([]WiFiHotspot, error) {
	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)

	log.Printf("开始使用nmcli扫描接口 %s 的WiFi热点...", interfaceName)

	args := []string{
		"-t", "-f", "SSID,SIGNAL,SECURITY,BSSID,CHAN",
		"device", "wifi", "list",
		fmt.Sprintf("ifname=%s", interfaceName),
	}
	cmd := exec.Command("nmcli", args...)
	log.Printf("执行命令: nmcli %v", args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("nmcli扫描失败: %v，将尝试使用iwlist", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("nmcli错误输出: %s", string(exitErr.Stderr))
		}
		return s.scanWiFiLinuxIwlist(interfaceName)
	}

	rawOutput := string(out)
	log.Printf("nmcli扫描原始输出(前100字符): %q...", safeSubstring(rawOutput, 100))
	if len(rawOutput) > 1000 {
		log.Printf("完整输出已记录到调试日志")
	}

	hotspots, err = parseNmcliOutput(rawOutput)
	if err != nil {
		log.Printf("解析nmcli输出失败: %v", err)
		return nil, fmt.Errorf("解析nmcli输出失败: %v", err)
	}

	log.Printf("nmcli扫描完成，发现 %d 个热点", len(hotspots))
	return hotspots, nil
}

func (s *NetworkService) scanWiFiLinuxIwlist(interfaceName string) ([]WiFiHotspot, error) {
	log.Printf("开始使用iwlist扫描接口 %s 的WiFi热点...", interfaceName)

	cmd := exec.Command("iwlist", interfaceName, "scan")
	log.Printf("执行命令: iwlist %s scan", interfaceName)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("iwlist扫描失败: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("iwlist错误输出: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("扫描WiFi失败: %v", err)
	}

	rawOutput := string(out)
	log.Printf("iwlist扫描原始输出(前100字符): %q...", safeSubstring(rawOutput, 100))
	if len(rawOutput) > 1000 {
		log.Printf("完整输出已记录到调试日志")
	}

	hotspots, err := parseIwlistOutput(rawOutput)
	if err != nil {
		log.Printf("解析iwlist输出失败: %v", err)
		return nil, fmt.Errorf("解析iwlist输出失败: %v", err)
	}

	log.Printf("iwlist扫描完成，发现 %d 个热点", len(hotspots))
	return hotspots, nil
}

// 解析netsh命令输出 (Windows)
func parseNetshOutput(output string) ([]WiFiHotspot, error) {
	log.Printf("开始解析WiFi扫描结果...")
	startTime := time.Now()
	defer func() {
		log.Printf("WiFi扫描结果解析完成，耗时: %v", time.Since(startTime))
	}()

	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)
	var currentHotspot *WiFiHotspot
	var parseErrors int

	lines := strings.Split(output, "\n")
	log.Printf("需要解析 %d 行输出", len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测新SSID开始 (处理中英文标签)
		if strings.HasPrefix(line, "SSID") || strings.HasPrefix(line, "SSID 名称") {
			if currentHotspot != nil {
				hotspots = append(hotspots, *currentHotspot)
				log.Printf("完成解析热点: %s (信号: %d%%, 加密: %s)",
					currentHotspot.SSID, currentHotspot.SignalStrength, currentHotspot.Security)
			}
			currentHotspot = &WiFiHotspot{}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.SSID = strings.TrimSpace(parts[1])
				log.Printf("发现新热点: %s (行 %d)", currentHotspot.SSID, i+1)
			} else {
				log.Printf("警告: 无法解析SSID行: %q", line)
				parseErrors++
			}
			continue
		}

		if currentHotspot == nil {
			continue
		}

		// 解析信号强度 (处理中英文标签)
		if strings.HasPrefix(line, "Signal") || strings.HasPrefix(line, "信号") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				percentStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "%"))
				if signal, err := strconv.Atoi(percentStr); err == nil {
					currentHotspot.SignalStrength = signal
				} else {
					log.Printf("警告: 无效的信号强度值: %q (行 %d)", parts[1], i+1)
					parseErrors++
				}
			}
		}

		// 解析加密类型 (处理中英文标签)
		if strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "身份验证") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.Security = strings.TrimSpace(parts[1])
			}
		}

		// 解析BSSID (处理中英文标签)
		if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.BSSID = strings.TrimSpace(parts[1])
			}
		}

		// 解析信道 (处理中英文标签)
		if strings.HasPrefix(line, "Channel") || strings.HasPrefix(line, "频道") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				if channel, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					currentHotspot.Channel = channel
				} else {
					log.Printf("警告: 无效的信道值: %q (行 %d)", parts[1], i+1)
					parseErrors++
				}
			}
		}
	}

	// 添加最后一个热点
	if currentHotspot != nil {
		hotspots = append(hotspots, *currentHotspot)
		log.Printf("完成解析热点: %s (信号: %d%%, 加密: %s)",
			currentHotspot.SSID, currentHotspot.SignalStrength, currentHotspot.Security)
	}

	// 过滤掉无效热点
	var validHotspots []WiFiHotspot
	var skipped int
	for _, hotspot := range hotspots {
		if hotspot.SSID != "" {
			validHotspots = append(validHotspots, hotspot)
		} else {
			skipped++
		}
	}

	log.Printf("解析完成: 共 %d 个热点(有效 %d 个，跳过 %d 个)，解析错误 %d 处",
		len(hotspots), len(validHotspots), skipped, parseErrors)
	return validHotspots, nil
}

// 解析nmcli命令输出 (Linux)
func parseNmcliOutput(output string) ([]WiFiHotspot, error) {
	log.Printf("开始解析nmcli输出...")
	startTime := time.Now()
	defer func() {
		log.Printf("nmcli输出解析完成，耗时: %v", time.Since(startTime))
	}()

	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)
	var parseErrors int

	lines := strings.Split(output, "\n")
	log.Printf("需要解析 %d 行nmcli输出", len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// nmcli -t 输出格式: SSID:SIGNAL:SECURITY:BSSID:CHAN
		fields := strings.Split(line, ":")
		if len(fields) < 5 {
			log.Printf("警告: 行 %d 字段不足(需要5个，得到%d个): %q",
				i+1, len(fields), line)
			parseErrors++
			continue
		}

		hotspot := WiFiHotspot{
			SSID:     fields[0],
			Security: fields[2],
			BSSID:    fields[3],
		}

		// 解析信号强度
		if signal, err := strconv.Atoi(fields[1]); err == nil {
			hotspot.SignalStrength = signal
		} else {
			log.Printf("警告: 行 %d 无效的信号强度值: %q", i+1, fields[1])
			parseErrors++
		}

		// 解析信道
		if channel, err := strconv.Atoi(fields[4]); err == nil {
			hotspot.Channel = channel
		} else {
			log.Printf("警告: 行 %d 无效的信道值: %q", i+1, fields[4])
			parseErrors++
		}

		log.Printf("解析热点: %s (信号: %d%%, 加密: %s)",
			hotspot.SSID, hotspot.SignalStrength, hotspot.Security)
		hotspots = append(hotspots, hotspot)
	}

	log.Printf("解析完成: 共 %d 个热点，解析错误 %d 处",
		len(hotspots), parseErrors)
	return hotspots, nil
}

// 解析iwlist命令输出 (Linux)
func parseIwlistOutput(output string) ([]WiFiHotspot, error) {
	log.Printf("开始解析iwlist输出...")
	startTime := time.Now()
	defer func() {
		log.Printf("iwlist输出解析完成，耗时: %v", time.Since(startTime))
	}()

	// 初始化空切片，确保不返回nil
	hotspots := make([]WiFiHotspot, 0)
	var currentHotspot *WiFiHotspot
	var parseErrors int
	var cellCount int

	lines := strings.Split(output, "\n")
	log.Printf("需要解析 %d 行iwlist输出", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测新Cell开始
		if strings.HasPrefix(line, "Cell") {
			cellCount++
			if currentHotspot != nil {
				hotspots = append(hotspots, *currentHotspot)
				log.Printf("完成解析热点: %s (信号: %d%%, 加密: %s)",
					currentHotspot.SSID, currentHotspot.SignalStrength, currentHotspot.Security)
			}
			currentHotspot = &WiFiHotspot{}
			continue
		}

		if currentHotspot == nil {
			continue
		}

		// 解析ESSID
		if strings.HasPrefix(line, "ESSID:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.SSID = strings.Trim(strings.TrimSpace(parts[1]), `"`)
				log.Printf("发现新热点: %s (Cell %d)", currentHotspot.SSID, cellCount)
			}
		}

		// 解析信号质量
		if strings.Contains(line, "Quality=") && strings.Contains(line, "Signal level=") {
			// 示例: Quality=70/70  Signal level=-40 dBm
			if parts := strings.Split(line, "Signal level="); len(parts) > 1 {
				signalParts := strings.Split(parts[1], " ")
				if len(signalParts) > 0 {
					// 将dBm转换为百分比 (近似)
					if dbm, err := strconv.Atoi(strings.TrimSpace(signalParts[0])); err == nil {
						// -30dBm ~ 100%, -90dBm ~ 0%
						currentHotspot.SignalStrength = clamp((dbm+90)*100/60, 0, 100)
					}
				}
			}
		}

		// 解析加密类型
		if strings.Contains(line, "Encryption key:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				if strings.TrimSpace(parts[1]) == "on" {
					// 默认加密类型
					currentHotspot.Security = "WPA2"
				} else {
					currentHotspot.Security = "Open"
				}
			}
		}

		// 解析MAC地址
		if strings.HasPrefix(line, "Address:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.BSSID = strings.TrimSpace(parts[1])
			}
		}

		// 解析信道
		if strings.HasPrefix(line, "Channel:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				if channel, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					currentHotspot.Channel = channel
				}
			}
		}
	}

	// 添加最后一个热点
	if currentHotspot != nil {
		hotspots = append(hotspots, *currentHotspot)
		log.Printf("完成解析热点: %s (信号: %d%%, 加密: %s)",
			currentHotspot.SSID, currentHotspot.SignalStrength, currentHotspot.Security)
	}

	log.Printf("解析完成: 共 %d 个Cell，有效热点 %d 个，解析错误 %d 处",
		cellCount, len(hotspots), parseErrors)
	return hotspots, nil
}

// clamp 确保值在[min,max]范围内
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (s *NetworkService) ConnectWiFi(interfaceName, ssid, password string) error {
	// 验证网卡是否存在且是无线网卡
	iface, err := s.GetInterface(interfaceName)
	if err != nil {
		return fmt.Errorf("网卡不存在: %v", err)
	}

	if iface.Hardware.AdapterType != "wireless" {
		return fmt.Errorf("网卡%s不是无线网卡", interfaceName)
	}

	// 根据操作系统执行不同命令
	switch runtime.GOOS {
	case "windows":
		return s.connectWiFiWindows(interfaceName, ssid, password)
	case "linux":
		return s.connectWiFiLinux(interfaceName, ssid, password)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (s *NetworkService) connectWiFiWindows(interfaceName, ssid, password string) error {
	// 记录原始SSID用于日志
	originalSSID := ssid

	// 确保SSID使用正确的编码
	log.Printf("处理WiFi连接请求，原始SSID: %q", ssid)

	// 检查SSID是否是URL编码的形式，如果是则进行解码
	if strings.Contains(ssid, "%") {
		decodedSSID, err := url.QueryUnescape(ssid)
		if err != nil {
			log.Printf("URL解码SSID失败: %v，将继续使用原始SSID", err)
		} else {
			ssid = decodedSSID
			log.Printf("URL解码后的SSID: %q", ssid)
		}
	}

	// 使用DecodeToUTF8确保SSID是UTF-8编码
	ssidBytes := []byte(ssid)
	decodedSSID, err := DecodeToUTF8(ssidBytes)
	if err != nil {
		log.Printf("SSID编码转换失败: %v，将使用当前SSID", err)
	} else {
		ssid = string(decodedSSID)
		log.Printf("编码转换后的SSID: %q", ssid)
	}

	// 检查解码后的SSID是否仍然包含URL编码字符，如果包含则可能是多次编码
	if strings.Contains(ssid, "%") {
		log.Printf("SSID仍包含URL编码字符，尝试再次解码")
		decodedSSID, err := url.QueryUnescape(ssid)
		if err != nil {
			log.Printf("二次URL解码失败: %v", err)
		} else {
			ssid = decodedSSID
			log.Printf("二次URL解码后的SSID: %q", ssid)
		}
	}

	// 先扫描可用的WiFi网络
	log.Printf("开始扫描可用的WiFi网络...")
	scanCmd := exec.Command("netsh", "wlan", "show", "networks")

	scanOutput, err := scanCmd.CombinedOutput()
	if err != nil {
		log.Printf("扫描WiFi网络失败: %v, 输出: %s", err, string(scanOutput))
		return fmt.Errorf("扫描WiFi网络失败: %v", err)
	}

	// 将扫描输出转换为UTF-8编码
	decodedOutput, err := DecodeToUTF8(scanOutput)
	if err != nil {
		log.Printf("转换扫描输出编码失败: %v", err)
	}
	scanOutputStr := string(decodedOutput)
	log.Printf("WiFi扫描原始输出:\n%s", scanOutputStr)

	// 检查SSID是否在可用网络列表中
	log.Printf("开始检查目标网络 %q 是否在可用列表中...", ssid)
	available := false
	var foundNetworks []string

	// 使用正则表达式提取SSID
	ssidRegex := regexp.MustCompile(`SSID\s+\d+\s*:\s*(.+)`)
	matches := ssidRegex.FindAllStringSubmatch(scanOutputStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			networkSSID := strings.TrimSpace(match[1])
			// 如果SSID被引号包围，去除引号
			networkSSID = strings.Trim(networkSSID, "\"")
			foundNetworks = append(foundNetworks, networkSSID)
			log.Printf("发现网络: %q (原始格式)", networkSSID)

			// 尝试不同的编码方式进行比较
			if networkSSID == ssid {
				available = true
				log.Printf("找到完全匹配的目标网络: %q", ssid)
				break
			}
		}
	}

	if !available {
		log.Printf("目标WiFi网络 %q 不在可用范围内", ssid)
		log.Printf("可用网络列表: %v", foundNetworks)
		log.Printf("请检查网络名称是否正确，以及网络是否在范围内")
		return fmt.Errorf("WiFi网络 %q 不在可用范围内", ssid)
	}

	log.Printf("目标网络 %q 在可用范围内，准备连接...", ssid)

	// 构建连接命令，使用双引号包围SSID以处理特殊字符
	cmd := exec.Command("netsh", "wlan", "connect",
		fmt.Sprintf("name=\"%s\"", ssid),
		fmt.Sprintf("interface=%s", interfaceName))

	if password != "" {
		log.Printf("WiFi需要密码，创建配置文件")

		// 先删除已有配置文件，不使用双引号，直接使用解码后的SSID
		deleteCmd := exec.Command("netsh", "wlan", "delete", "profile",
			fmt.Sprintf("name=%s", ssid),
			fmt.Sprintf("interface=%s", interfaceName))
		if out, err := deleteCmd.CombinedOutput(); err != nil {
			log.Printf("删除旧配置文件失败(可能不存在): %s", string(out))
		}

		// 对XML中的特殊字符进行转义
		xmlEscapedSSID := html.EscapeString(ssid)
		xmlEscapedPassword := html.EscapeString(password)

		log.Printf("XML转义后的SSID: %q", xmlEscapedSSID)

		// 创建XML配置文件，确保使用UTF-8编码
		// 使用更灵活的安全设置，支持多种加密类型
		profile := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
	<name>%s</name>
	<SSIDConfig>
		<SSID>
			<hex>%s</hex>
			<name>%s</name>
		</SSID>
		<nonBroadcast>false</nonBroadcast>
	</SSIDConfig>
	<connectionType>ESS</connectionType>
	<connectionMode>auto</connectionMode>
	<autoSwitch>false</autoSwitch>
	<MSM>
		<security>
			<authEncryption>
				<authentication>WPA2PSK</authentication>
				<encryption>AES</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
			<sharedKey>
				<keyType>passPhrase</keyType>
				<protected>false</protected>
				<keyMaterial>%s</keyMaterial>
			</sharedKey>
		</security>
	</MSM>
	<MacRandomization xmlns="http://www.microsoft.com/networking/WLAN/profile/v3">
		<enableRandomization>false</enableRandomization>
	</MacRandomization>
</WLANProfile>`, xmlEscapedSSID, bytesToHexString([]byte(ssid)), xmlEscapedSSID, xmlEscapedPassword)

		log.Printf("生成的WiFi配置文件内容:\n%s", profile)

		// 写入临时文件，确保使用UTF-8编码
		tmpFile, err := os.CreateTemp("", "wifi_*.xml")
		if err != nil {
			return fmt.Errorf("创建临时文件失败: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		// 写入UTF-8 BOM标记，确保Windows正确识别UTF-8编码
		utf8BOM := []byte{0xEF, 0xBB, 0xBF}
		if _, err := tmpFile.Write(utf8BOM); err != nil {
			return fmt.Errorf("写入UTF-8 BOM失败: %v", err)
		}

		if _, err := tmpFile.WriteString(profile); err != nil {
			return fmt.Errorf("写入配置文件失败: %v", err)
		}
		tmpFile.Close()

		log.Printf("WiFi配置文件已创建: %s", tmpFile.Name())

		// 添加配置文件
		addCmd := exec.Command("netsh", "wlan", "add", "profile",
			fmt.Sprintf("filename=%s", tmpFile.Name()),
			fmt.Sprintf("interface=%s", interfaceName))

		// 设置命令环境变量，确保正确处理UTF-8
		addCmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

		addOutput, err := addCmd.CombinedOutput()
		if err != nil {
			log.Printf("添加配置文件失败，输出: %s", string(addOutput))

			// 尝试使用备用方法添加配置文件
			log.Printf("尝试使用备用方法添加配置文件...")
			addCmd2 := exec.Command("netsh", "wlan", "add", "profile",
				fmt.Sprintf("filename=\"%s\"", tmpFile.Name()))

			addOutput2, err2 := addCmd2.CombinedOutput()
			if err2 != nil {
				log.Printf("备用方法添加配置文件也失败，输出: %s", string(addOutput2))
				return fmt.Errorf("添加配置文件失败: %s, %v", string(addOutput2), err2)
			}

			log.Printf("备用方法成功添加WiFi配置文件")
		} else {
			log.Printf("WiFi配置文件已添加，输出: %s", string(addOutput))
		}
	}

	// 尝试使用不同的连接方法
	log.Printf("尝试方法1: 使用netsh wlan connect命令连接...")
	log.Printf("执行WiFi连接命令: netsh wlan connect name=\"%s\" interface=%s", ssid, interfaceName)

	// 设置命令环境变量，确保正确处理UTF-8
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("方法1连接失败，输出: %s", string(out))

		// 尝试方法2: 使用ssid=代替name=
		log.Printf("尝试方法2: 使用ssid=参数代替name=...")
		cmd2 := exec.Command("netsh", "wlan", "connect",
			fmt.Sprintf("ssid=\"%s\"", ssid),
			fmt.Sprintf("interface=%s", interfaceName))
		cmd2.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			log.Printf("方法2连接失败，输出: %s", string(out2))

			// 尝试方法3: 不使用引号
			log.Printf("尝试方法3: 不使用引号包围SSID...")
			cmd3 := exec.Command("netsh", "wlan", "connect",
				fmt.Sprintf("name=%s", ssid),
				fmt.Sprintf("interface=%s", interfaceName))
			cmd3.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

			out3, err3 := cmd3.CombinedOutput()
			if err3 != nil {
				log.Printf("方法3连接失败，输出: %s", string(out3))
				return fmt.Errorf("所有连接方法均失败，最后错误: %s, %v", string(out3), err3)
			}

			log.Printf("方法3连接成功")
			return nil
		}

		log.Printf("方法2连接成功")
		return nil
	}

	log.Printf("方法1连接成功，原始SSID: %q", originalSSID)
	return nil
}

func (s *NetworkService) connectWiFiLinux(interfaceName, ssid, password string) error {
	// Linux实现使用nmcli
	var cmd *exec.Cmd
	if password == "" {
		cmd = exec.Command("nmcli", "device", "wifi", "connect",
			ssid,
			"ifname", interfaceName)
	} else {
		cmd = exec.Command("nmcli", "device", "wifi", "connect",
			ssid,
			"password", password,
			"ifname", interfaceName)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("连接失败: %s, %v", string(out), err)
	}

	return nil
}

func (s *NetworkService) CheckConnectivity(target string) (models.ConnectivityResult, error) {
	if target == "" {
		target = "http://www.baidu.com" // 默认探测百度
	}

	log.Printf("开始检查网络连通性，目标: %s", target)

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(target)
	duration := time.Since(start)

	result := models.ConnectivityResult{
		Target:     target,
		DurationMs: duration.Milliseconds(),
	}

	if err != nil {
		log.Printf("网络连通性检查失败: %v", err)
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	log.Printf("网络连通性检查成功，状态码: %d, 耗时: %dms", resp.StatusCode, duration.Milliseconds())

	result.Success = true
	result.StatusCode = resp.StatusCode
	return result, nil
}

// GetAvailableWiFiHotspots 获取指定WIFI网卡可连接的热点列表
// getConnectedSSID 获取无线网卡当前连接的SSID
func getConnectedSSID(interfaceName string) (string, error) {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces", "interface="+interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取SSID失败: %v", err)
	}

	// 解析输出查找SSID行
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "SSID") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", nil // 没有连接热点时返回空
}

func (s *NetworkService) GetAvailableWiFiHotspots(interfaceName string) ([]models.WiFiHotspot, error) {
	log.Printf("开始获取接口 %s 的可用WIFI热点列表", interfaceName)

	// 执行netsh命令获取热点列表
	cmd := exec.Command("netsh", "wlan", "show", "networks", "interface="+interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("获取WIFI热点列表失败: %v, 输出: %s", err, string(output))
		return []models.WiFiHotspot{}, fmt.Errorf("获取WIFI热点列表失败: %v", err)
	}

	// 解析命令输出
	hotspots := parseWiFiHotspots(string(output))
	log.Printf("成功获取 %d 个WIFI热点", len(hotspots))
	return hotspots, nil
}

// parseWiFiHotspots 解析netsh命令输出的WIFI热点信息
func parseWiFiHotspots(output string) []models.WiFiHotspot {
	var hotspots []models.WiFiHotspot
	var currentHotspot *models.WiFiHotspot

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测新热点开始
		if strings.HasPrefix(line, "SSID") {
			if currentHotspot != nil {
				hotspots = append(hotspots, *currentHotspot)
			}
			currentHotspot = &models.WiFiHotspot{}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.SSID = strings.TrimSpace(parts[1])
			}
			continue
		}

		if currentHotspot == nil {
			continue
		}

		// 解析其他热点属性
		if strings.HasPrefix(line, "Network type") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.RadioType = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Authentication") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.SecurityType = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				percent := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
				if signal, err := strconv.Atoi(percent); err == nil {
					currentHotspot.SignalLevel = signal
				}
			}
		} else if strings.HasPrefix(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				if channel, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					currentHotspot.Channel = channel
				}
			}
		} else if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentHotspot.BSSID = strings.TrimSpace(parts[1])
			}
		}
	}

	// 添加最后一个热点
	if currentHotspot != nil {
		hotspots = append(hotspots, *currentHotspot)
	}

	return hotspots
}
