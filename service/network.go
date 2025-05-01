package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"networkconfig/models"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NetworkService 处理网络配置相关的操作
type NetworkService struct{}

// NewNetworkService 创建新的NetworkService实例
func NewNetworkService() *NetworkService {
	return &NetworkService{}
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

		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			log.Printf("跳过回环接口: %s", iface.Name)
			continue
		}

		// 跳过未启用的接口
		if iface.Flags&net.FlagUp == 0 {
			log.Printf("跳过未启用的接口: %s", iface.Name)
			continue
		}

		ifaceInfo, err := s.GetInterface(iface.Name)
		if err != nil {
			log.Printf("获取接口 %s 信息失败: %v", iface.Name, err)
			// 创建基本接口信息
			basicInfo := models.Interface{
				Name:        iface.Name,
				Description: iface.Name,
				Status:      getInterfaceStatus(iface.Flags),
				Hardware: models.Hardware{
					MACAddress: iface.HardwareAddr.String(),
				},
				Driver: models.Driver{
					Name: iface.Name,
				},
			}
			interfaces = append(interfaces, basicInfo)
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

	// 获取硬件和驱动信息（即使失败也继续）
	hardware, err := getHardwareInfo(name)
	if err != nil {
		log.Printf("获取接口 %s 硬件信息失败: %v", name, err)
		ifaceInfo.Hardware = models.Hardware{
			MACAddress: iface.HardwareAddr.String(),
		}
	} else {
		ifaceInfo.Hardware = hardware
		log.Printf("接口 %s 硬件信息: %+v", name, hardware)
	}

	driver, err := getDriverInfo(name)
	if err != nil {
		log.Printf("获取接口 %s 驱动信息失败: %v", name, err)
		ifaceInfo.Driver = models.Driver{
			Name: iface.Name,
		}
	} else {
		ifaceInfo.Driver = driver
		log.Printf("接口 %s 驱动信息: %+v", name, driver)
	}

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

	return models.Hardware{
		MACAddress:    result.MACAddress,
		Manufacturer:  result.Manufacturer,
		ProductName:   result.ProductName,
		AdapterType:   result.AdapterType,
		PhysicalMedia: "Ethernet", // 默认值，可以根据实际情况修改
		Speed:         speedStr,
		BusType:       result.AdapterType,
		PNPDeviceID:   result.PNPDeviceID,
	}, nil
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

		// 设置DHCP自动获取IP
		log.Printf("为接口 %s 设置DHCP自动获取IP", name)

		cmd := exec.Command("netsh",
			"interface",
			"ipv4",
			"set",
			"address",
			fmt.Sprintf("name=%s", name), // 直接传递接口名称，无需引号
			"source=dhcp")

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("设置DHCP失败: %v, 输出: %s", err, string(output))
			return fmt.Errorf("设置DHCP失败: %v, 输出: %s", err, string(output))
		}
		log.Printf("成功设置DHCP自动获取IP")

		// 设置DNS
		if config.DNSAuto {
			cmdStr := fmt.Sprintf("netsh interface ipv4 set dnsservers name=\"%s\" source=dhcp", name)
			log.Printf("执行命令: %s", cmdStr)

			cmd = exec.Command("netsh", "interface", "ipv4", "set", "dnsservers",
				fmt.Sprintf(`name=\"%s\"`, name),
				"source=dhcp")

			output, err = cmd.CombinedOutput()
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
						fmt.Sprintf(`name=\"%s\"`, name),
						"static",
						dns)
				} else {
					cmdStr = fmt.Sprintf("netsh interface ipv4 add dns name=\"%s\" %s index=%d",
						name, dns, i+1)
					cmd = exec.Command("netsh", "interface", "ipv4", "add", "dns",
						fmt.Sprintf(`name=\"%s\"`, name),
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
						fmt.Sprintf(`name="%s"`, name),
						"static",
						dns)
				} else {
					cmdStr = fmt.Sprintf("netsh interface ipv4 add dns name=\"%s\" %s index=%d",
						name, dns, i+1)
					cmd = exec.Command("netsh", "interface", "ipv4", "add", "dns",
						fmt.Sprintf(`name="%s"`, name),
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
		fmt.Sprintf("store=persistent"))

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

func parseDNSServers(output string) []string {
	// 简单实现，实际使用时需要更复杂的解析逻辑
	var servers []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "DNS servers") {
			fields := strings.Fields(line)
			if len(fields) > 2 {
				servers = append(servers, fields[2])
			}
		}
	}
	return servers
}
