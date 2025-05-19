package service

import (
	"encoding/json"
	"fmt"
	"log"
	"networkconfig/models"
	"os/exec"
	"strings"
)

// Win11HotspotManager 管理Windows 11移动热点
type Win11HotspotManager struct {
	debug bool
}

// NewWin11HotspotManager 创建新的Windows 11热点管理器
func NewWin11HotspotManager(debug bool) *Win11HotspotManager {
	return &Win11HotspotManager{
		debug: debug,
	}
}

// GetStatus 获取Windows 11移动热点状态
func (m *Win11HotspotManager) GetStatus() (models.HotspotStatus, error) {
	// PowerShell命令获取移动热点状态
	psCmd := `
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$PSDefaultParameterValues['*:Encoding'] = 'utf8'
		
		try {
			Add-Type -AssemblyName Windows.Networking
			$tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile())
			
			@{
				Enabled = $tetheringManager.TetheringOperationalState -eq 'On'
				SSID = $tetheringManager.GetCurrentAccessPointConfiguration().Ssid
				MaxClients = $tetheringManager.GetCurrentAccessPointConfiguration().MaxClientCount
				Authentication = $tetheringManager.GetCurrentAccessPointConfiguration().Authentication
				Encryption = $tetheringManager.GetCurrentAccessPointConfiguration().Encryption
				ClientsCount = ($tetheringManager.GetTetheringClients() | Measure-Object).Count
			} | ConvertTo-Json
		} catch {
			Write-Error $_.Exception.Message
			exit 1
		}
	`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		if m.debug {
			log.Printf("获取热点状态失败: %v, 输出: %s", err, string(output))
		}
		return models.HotspotStatus{}, fmt.Errorf("获取热点状态失败: %v", err)
	}

	// 解析JSON输出
	var result struct {
		Enabled        bool   `json:"Enabled"`
		SSID           string `json:"SSID"`
		MaxClients     int    `json:"MaxClients"`
		Authentication string `json:"Authentication"`
		Encryption     string `json:"Encryption"`
		ClientsCount   int    `json:"ClientsCount"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return models.HotspotStatus{}, fmt.Errorf("解析热点状态失败: %v", err)
	}

	return models.HotspotStatus{
		Enabled:        result.Enabled,
		SSID:           result.SSID,
		MaxClients:     result.MaxClients,
		Authentication: result.Authentication,
		Encryption:     result.Encryption,
		ClientsCount:   result.ClientsCount,
	}, nil
}

// Configure 配置Windows 11移动热点
func (m *Win11HotspotManager) Configure(config models.HotspotConfig) error {
	if config.SSID == "" {
		return fmt.Errorf("SSID不能为空")
	}
	if len(config.Password) < 8 {
		return fmt.Errorf("密码长度必须至少为8个字符")
	}

	// PowerShell命令配置移动热点
	psCmd := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$PSDefaultParameterValues['*:Encoding'] = 'utf8'
		
		try {
			Add-Type -AssemblyName Windows.Networking
			$tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile())
			
			$config = New-Object Windows.Networking.NetworkOperators.NetworkOperatorTetheringAccessPointConfiguration
			$config.Ssid = '%s'
			$config.Passphrase = '%s'
			
			$tetheringManager.ConfigureAccessPointAsync($config).AsTask().Wait()
			
			if (%t) {
				$tetheringManager.StartTetheringAsync().AsTask().Wait()
			}
			
			Write-Output "Success"
		} catch {
			Write-Error $_.Exception.Message
			exit 1
		}
	`, config.SSID, config.Password, config.Enabled)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if m.debug {
			log.Printf("配置热点失败: %v, 输出: %s", err, string(output))
		}
		return fmt.Errorf("配置热点失败: %v", err)
	}

	if !strings.Contains(string(output), "Success") {
		return fmt.Errorf("配置热点失败: %s", string(output))
	}

	return nil
}

// SetStatus 启用或禁用Windows 11移动热点
func (m *Win11HotspotManager) SetStatus(enable bool) error {
	// PowerShell命令启用/禁用移动热点
	psCmd := fmt.Sprintf(`
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		$PSDefaultParameterValues['*:Encoding'] = 'utf8'
		
		try {
			Add-Type -AssemblyName Windows.Networking
			$tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile())
			
			if (%t) {
				$tetheringManager.StartTetheringAsync().AsTask().Wait()
			} else {
				$tetheringManager.StopTetheringAsync().AsTask().Wait()
			}
			
			Write-Output "Success"
		} catch {
			Write-Error $_.Exception.Message
			exit 1
		}
	`, enable)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if m.debug {
			log.Printf("修改热点状态失败: %v, 输出: %s", err, string(output))
		}
		return fmt.Errorf("修改热点状态失败: %v", err)
	}

	if !strings.Contains(string(output), "Success") {
		return fmt.Errorf("修改热点状态失败: %s", string(output))
	}

	return nil
}