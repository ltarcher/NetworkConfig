package service

import (
	"encoding/json"
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

	cmd := exec.Command("powershell", "-Command", "[System.Environment]::OSVersion.Version | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	var version struct {
		Major    int `json:"Major"`
		Minor    int `json:"Minor"`
		Build    int `json:"Build"`
		Revision int `json:"Revision"`
	}
	if err := json.Unmarshal(output, &version); err != nil {
		return false
	}

	// Windows 11的版本号是10.0.22000或更高
	return version.Major == 10 && version.Build >= 22000
}

// runHotspotDiagnostic 运行热点诊断
func (s *NetworkService) runHotspotDiagnostic() {
	log.Println("运行热点诊断...")

	// 运行诊断命令
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-File", "hotspot.ps1", "diagnostic")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("运行热点诊断失败: %v", err)
		return
	}

	// 输出诊断信息
	log.Printf("热点诊断信息: %s", string(output))

	// 检查系统环境
	s.checkSystemEnvironment()
}

// checkSystemEnvironment 检查系统环境
func (s *NetworkService) checkSystemEnvironment() {
	// 检查PowerShell执行策略
	cmd := exec.Command("powershell", "-Command", "Get-ExecutionPolicy")
	output, err := cmd.CombinedOutput()
	if err == nil {
		policy := strings.TrimSpace(string(output))
		log.Printf("PowerShell执行策略: %s", policy)
		if policy == "Restricted" {
			log.Println("警告: PowerShell执行策略为Restricted，可能影响热点管理功能")
		}
	}

	// 检查网络适配器状态
	cmd = exec.Command("powershell", "-Command", "Get-NetAdapter | Where-Object { $_.Status -eq 'Up' } | ConvertTo-Json")
	output, err = cmd.CombinedOutput()
	if err == nil {
		log.Printf("活动网络适配器: %s", string(output))
	}

	// 检查移动热点服务
	cmd = exec.Command("powershell", "-Command", "Get-Service -Name SharedAccess | Select-Object Name, Status | ConvertTo-Json")
	output, err = cmd.CombinedOutput()
	if err == nil {
		log.Printf("Internet连接共享服务状态: %s", string(output))
	}
}

// GetHotspotStatus 获取移动热点状态
func (s *NetworkService) GetHotspotStatus() (models.HotspotStatus, error) {
	if isWin11OrLater() {
		manager := NewWin11HotspotManager(s.Debug)
		status, err := manager.GetStatus()
		if err != nil && s.Debug {
			log.Printf("Windows 11 API获取热点状态失败: %v, 尝试运行诊断", err)
			s.runHotspotDiagnostic()

			// 尝试使用netsh命令作为备选方案
			log.Println("尝试使用netsh命令获取热点状态...")
			return s.getHotspotStatusWithNetsh()
		}
		return status, err
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	return s.getHotspotStatusWithNetsh()
}

// getHotspotStatusWithNetsh 使用netsh命令获取热点状态
func (s *NetworkService) getHotspotStatusWithNetsh() (models.HotspotStatus, error) {
	log.Printf("开始获取移动热点状态...")

	cmd := exec.Command("netsh", "wlan", "show", "hostednetwork")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("获取移动热点状态失败: %v", err)
		if s.Debug {
			s.runHotspotDiagnostic()
		}
		return models.HotspotStatus{}, fmt.Errorf("获取热点状态失败: %v", err)
	}

	// 解析输出
	status := models.HotspotStatus{
		Success: true, // 如果能执行到这里，说明命令执行成功
	}
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
					status.MaxClientCount = maxClients
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
		err := manager.Configure(config)
		if err != nil && s.Debug {
			log.Printf("Windows 11 API配置热点失败: %v, 尝试运行诊断", err)
			s.runHotspotDiagnostic()

			// 尝试使用netsh命令作为备选方案
			log.Println("尝试使用netsh命令配置热点...")
			return s.configureHotspotWithNetsh(config)
		}
		return err
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	return s.configureHotspotWithNetsh(config)
}

// configureHotspotWithNetsh 使用netsh命令配置热点
func (s *NetworkService) configureHotspotWithNetsh(config models.HotspotConfig) error {
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
		if s.Debug {
			s.runHotspotDiagnostic()
		}
		return fmt.Errorf("配置热点失败: %v", err)
	}

	// 如果需要启用热点
	if config.Enabled {
		if err := s.setHotspotStatusWithNetsh(true); err != nil {
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
		err := manager.SetStatus(enable)
		if err != nil && s.Debug {
			log.Printf("Windows 11 API设置热点状态失败: %v, 尝试运行诊断", err)
			s.runHotspotDiagnostic()

			// 尝试使用netsh命令作为备选方案
			log.Println("尝试使用netsh命令设置热点状态...")
			return s.setHotspotStatusWithNetsh(enable)
		}
		return err
	}

	// 对于Windows 10及更早版本，使用原有的netsh实现
	return s.setHotspotStatusWithNetsh(enable)
}

// setHotspotStatusWithNetsh 使用netsh命令设置热点状态
func (s *NetworkService) setHotspotStatusWithNetsh(enable bool) error {
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
		if s.Debug {
			s.runHotspotDiagnostic()
		}
		return fmt.Errorf("修改热点状态失败: %v", err)
	}

	log.Printf("成功%s移动热点", map[bool]string{true: "启用", false: "禁用"}[enable])
	return nil
}
