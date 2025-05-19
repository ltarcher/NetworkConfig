package service

import (
	"fmt"
	"log"
	"networkconfig/models"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// isWin11OrLater 检查是否是Windows 11或更高版本
func isWin11OrLater() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	cmd := exec.Command("powershell", "-Command", "Get-CimInstance Win32_OperatingSystem | Select-Object Version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	version := strings.TrimSpace(string(output))
	// Windows 11的版本号是10.0.22000或更高
	return strings.Contains(version, "10.0.22") || strings.Contains(version, "10.0.23")
}

// GetHotspotStatus 获取移动热点状态
func (s *NetworkService) GetHotspotStatus() (models.HotspotStatus, error) {
	if isWin11OrLater() {
		manager := NewWin11HotspotManager(s.Debug)
		return manager.GetStatus()
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	log.Printf("开始获取移动热点状态...")

	cmd := exec.Command("netsh", "wlan", "show", "hostednetwork")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("获取移动热点状态失败: %v", err)
		return models.HotspotStatus{}, fmt.Errorf("获取移动热点状态失败: %v", err)
	}

	// 解析输出
	status := models.HotspotStatus{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Status") || strings.Contains(line, "状态") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.Enabled = strings.TrimSpace(parts[1]) == "Started" ||
					strings.TrimSpace(parts[1]) == "已启动"
			}
		} else if strings.Contains(line, "SSID name") || strings.Contains(line, "SSID 名称") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.SSID = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		} else if strings.Contains(line, "Max number of clients") || strings.Contains(line, "最大客户端数") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if maxClients, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.MaxClients = maxClients
				}
			}
		} else if strings.Contains(line, "Authentication") || strings.Contains(line, "身份验证") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.Authentication = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Cipher") || strings.Contains(line, "加密") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.Encryption = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Number of clients") || strings.Contains(line, "客户端数") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if clientsCount, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.ClientsCount = clientsCount
				}
			}
		}
	}

	log.Printf("成功获取移动热点状态: %+v", status)
	return status, nil
}

// ConfigureHotspot 配置移动热点
func (s *NetworkService) ConfigureHotspot(config models.HotspotConfig) error {
	if isWin11OrLater() {
		manager := NewWin11HotspotManager(s.Debug)
		return manager.Configure(config)
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	log.Printf("开始配置移动热点: %+v", config)

	// 验证SSID和密码
	if config.SSID == "" {
		return fmt.Errorf("SSID不能为空")
	}
	if len(config.Password) < 8 {
		return fmt.Errorf("密码长度必须至少为8个字符")
	}

	// 设置热点配置
	cmd := exec.Command("netsh", "wlan", "set", "hostednetwork",
		fmt.Sprintf("mode=allow"),
		fmt.Sprintf("ssid=%s", config.SSID),
		fmt.Sprintf("key=%s", config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("配置移动热点失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("配置移动热点失败: %v", err)
	}

	// 如果需要启用热点
	if config.Enabled {
		if err := s.SetHotspotStatus(true); err != nil {
			return fmt.Errorf("启用移动热点失败: %v", err)
		}
	}

	log.Printf("成功配置移动热点")
	return nil
}

// SetHotspotStatus 启用或禁用移动热点
func (s *NetworkService) SetHotspotStatus(enable bool) error {
	if isWin11OrLater() {
		manager := NewWin11HotspotManager(s.Debug)
		return manager.SetStatus(enable)
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	var cmd *exec.Cmd
	if enable {
		log.Printf("正在启用移动热点...")
		cmd = exec.Command("netsh", "wlan", "start", "hostednetwork")
	} else {
		log.Printf("正在禁用移动热点...")
		cmd = exec.Command("netsh", "wlan", "stop", "hostednetwork")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("修改移动热点状态失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("修改移动热点状态失败: %v", err)
	}

	log.Printf("成功%s移动热点", map[bool]string{true: "启用", false: "禁用"}[enable])
	return nil
}