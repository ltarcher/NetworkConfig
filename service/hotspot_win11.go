package service

import (
	"encoding/json"
	"fmt"
	"networkconfig/models"
	"os/exec"
)

// Win11HotspotManager 管理Windows移动热点
type Win11HotspotManager struct {
	debug bool
}

// NewWin11HotspotManager 创建新的热点管理器
func NewWin11HotspotManager(debug bool) *Win11HotspotManager {
	return &Win11HotspotManager{
		debug: debug,
	}
}

// GetStatus 获取热点状态
func (m *Win11HotspotManager) GetStatus() (models.HotspotStatus, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-File", "hotspot.ps1", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return models.HotspotStatus{}, fmt.Errorf("获取热点状态失败: %v", err)
	}

	// 解析JSON输出
	var result struct {
		Success        bool   `json:"Success"`
		Error          string `json:"Error"`
		Enabled        bool   `json:"Enabled"`
		SSID           string `json:"SSID"`
		ClientsCount   int    `json:"ClientsCount"`
		Authentication string `json:"Authentication"`
		Encryption     string `json:"Encryption"`
		MaxClientCount int    `json:"MaxClientCount"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return models.HotspotStatus{}, fmt.Errorf("解析热点状态失败: %v", err)
	}

	if !result.Success {
		return models.HotspotStatus{}, fmt.Errorf("获取热点状态失败: %s", result.Error)
	}

	return models.HotspotStatus{
		Enabled:        result.Enabled,
		SSID:           result.SSID,
		MaxClients:     result.MaxClientCount,
		Authentication: result.Authentication,
		Encryption:     result.Encryption,
		ClientsCount:   result.ClientsCount,
	}, nil
}

// Configure 配置热点
func (m *Win11HotspotManager) Configure(config models.HotspotConfig) error {
	// 验证参数
	if config.SSID == "" {
		return fmt.Errorf("SSID不能为空")
	}
	if len(config.SSID) > 32 {
		return fmt.Errorf("SSID长度不能超过32个字符")
	}
	if config.Password == "" {
		return fmt.Errorf("密码不能为空")
	}
	if len(config.Password) < 8 || len(config.Password) > 63 {
		return fmt.Errorf("密码长度必须在8-63个字符之间")
	}

	args := []string{"-NoProfile", "-NonInteractive", "-File", "hotspot.ps1", "configure", "-SSID", config.SSID, "-Password", config.Password}
	if config.Enabled {
		args = append(args, "-Enable")
	}

	cmd := exec.Command("powershell", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("配置热点失败: %v", err)
	}

	// 解析JSON输出
	var result struct {
		Success bool   `json:"Success"`
		Error   string `json:"Error"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("解析配置结果失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("配置热点失败: %s", result.Error)
	}

	return nil
}

// SetStatus 设置热点状态
func (m *Win11HotspotManager) SetStatus(enable bool) error {
	action := "enable"
	if !enable {
		action = "disable"
	}

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-File", "hotspot.ps1", action)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("设置热点状态失败: %v", err)
	}

	// 解析JSON输出
	var result struct {
		Success bool   `json:"Success"`
		Error   string `json:"Error"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("解析状态变更结果失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("设置热点状态失败: %s", result.Error)
	}

	return nil
}
