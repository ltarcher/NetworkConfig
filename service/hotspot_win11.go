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
	// PowerShell通用代码块，包含Windows Runtime assemblies加载和辅助函数
	commonCode string
}

// 初始化PowerShell通用代码块
var psCommonCode = `
# Set output encoding to UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    @{
        Success = $false
        Error = "This script requires administrator privileges"
    } | ConvertTo-Json
    exit 1
}

# Load Windows Runtime assemblies
try {
    Add-Type -AssemblyName System.Runtime.WindowsRuntime
    
    # Helper function to await WinRT async operations
    function Await($Task, $ResultType) {
        $asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | 
            Where-Object { 
                $_.Name -eq 'AsTask' -and 
                $_.GetParameters().Count -eq 1 -and 
                $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncAction' 
            })[0]
        
        $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
        $netTask = $asTask.Invoke($null, @($Task))
        $netTask.Wait(-1) | Out-Null
    }

    # Get TetheringManager instance
    function Get-TetheringManager {
        try {
            $connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows.Networking.Connectivity,ContentType=WindowsRuntime]::GetInternetConnectionProfile()
            if ($null -eq $connectionProfile) {
                throw "No active internet connection found"
            }
            
            $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows.Networking.NetworkOperators,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile)
            if ($null -eq $tetheringManager) {
                throw "Failed to create tethering manager"
            }
            
            return $tetheringManager
        }
        catch {
            throw "Failed to initialize tethering manager: $_"
        }
    }
}
catch {
    @{
        Success = $false
        Error = "Failed to load Windows Runtime assemblies: $_"
    } | ConvertTo-Json
    exit 1
}
`

// NewWin11HotspotManager 创建新的热点管理器
func NewWin11HotspotManager(debug bool) *Win11HotspotManager {
	return &Win11HotspotManager{
		debug:      debug,
		commonCode: psCommonCode,
	}
}

// GetStatus 获取热点状态
func (m *Win11HotspotManager) GetStatus() (models.HotspotStatus, error) {
	// 构建PowerShell脚本内容
	psScript := fmt.Sprintf(`
%s
try {
    $tetheringManager = Get-TetheringManager
    
    # Get current configuration
    $config = $tetheringManager.GetCurrentAccessPointConfiguration()
    
    # Check if tethering is enabled
    $tetheringOperationalState = $tetheringManager.TetheringOperationalState
    $isEnabled = $tetheringOperationalState -eq 1  # 1 = On
    
    # Get client count
    $clientCount = 0
    if ($isEnabled) {
        try {
            $clients = $tetheringManager.GetTetheringClients()
            $clientCount = @($clients).Count
        }
        catch {
            # Ignore errors when getting clients
        }
    }
    
    # Return status as JSON
    @{
        Success = $true
        Enabled = $isEnabled
        SSID = $config.Ssid
        ClientsCount = $clientCount
        Authentication = "WPA2PSK"
        Encryption = "AES"
        MaxClientCount = $tetheringManager.MaxClientCount
    } | ConvertTo-Json
}
catch {
    @{
        Success = $false
        Error = $_.Exception.Message
    } | ConvertTo-Json
}
`, m.commonCode)

	// 执行PowerShell脚本
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
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

	// 构建PowerShell脚本内容
	psScript := fmt.Sprintf(`
%s
try {
    $tetheringManager = Get-TetheringManager
    
    # Configure the access point
    $configuration = $tetheringManager.GetCurrentAccessPointConfiguration()
    $configuration.Ssid = "%s"
    $configuration.Passphrase = "%s"
    
    # Apply the configuration
    $configureTask = $tetheringManager.ConfigureAccessPointAsync($configuration)
    Await $configureTask ([System.Threading.Tasks.VoidTaskResult])
    
    # Enable/disable the hotspot if requested
    if ($true -eq %v) {
        if ($tetheringManager.TetheringOperationalState -ne 1) {
            $enableTask = $tetheringManager.EnableAsync()
            Await $enableTask ([System.Threading.Tasks.VoidTaskResult])
        }
    }
    
    @{
        Success = $true
    } | ConvertTo-Json
}
catch {
    @{
        Success = $false
        Error = $_.Exception.Message
    } | ConvertTo-Json
}
`, m.commonCode, config.SSID, config.Password, config.Enabled)

	// 执行PowerShell脚本
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
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

// getActionWord 根据启用状态返回对应的动作词
func getActionWord(enable bool) string {
	if enable {
		return "Enable"
	}
	return "Disable"
}

// SetStatus 设置热点状态
func (m *Win11HotspotManager) SetStatus(enable bool) error {
	// 构建PowerShell脚本内容
	action := "EnableAsync"
	if !enable {
		action = "DisableAsync"
	}
	
	psScript := fmt.Sprintf(`
%s
try {
    $tetheringManager = Get-TetheringManager
    
    # %s the hotspot
    $task = $tetheringManager.%s()
    Await $task ([System.Threading.Tasks.VoidTaskResult])
    
    @{
        Success = $true
    } | ConvertTo-Json
}
catch {
    @{
        Success = $false
        Error = $_.Exception.Message
    } | ConvertTo-Json
}
`, m.commonCode, getActionWord(enable), action)

	// 执行PowerShell脚本
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
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