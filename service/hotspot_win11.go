package service

import (
	"encoding/json"
	"fmt"
	"networkconfig/models"
	"os/exec"
	"strings"
)

// Win11HotspotManager 管理Windows移动热点
type Win11HotspotManager struct {
	debug bool
	// PowerShell通用代码块，包含Windows Runtime assemblies加载和辅助函数
	commonCode string
	// 是否已初始化执行策略
	policyInitialized bool
}

// 设置PowerShell执行策略
func (m *Win11HotspotManager) setExecutionPolicy() error {
	if m.policyInitialized {
		return nil
	}

	// 首先检查当前执行策略
	checkCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-ExecutionPolicy -Scope CurrentUser")
	output, err := checkCmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), "RemoteSigned") {
		m.policyInitialized = true
		return nil
	}

	// 尝试设置执行策略
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned -Force")
	output, err = cmd.CombinedOutput()
	if err != nil {
		if m.debug {
			fmt.Printf("设置PowerShell执行策略失败: %v - %s\n", err, string(output))
		}
		return fmt.Errorf(`设置PowerShell执行策略失败: %v
当前执行策略为Restricted，需要管理员权限修改。
请使用管理员权限运行PowerShell并执行:
Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned`, err)
	}

	m.policyInitialized = true
	if m.debug {
		fmt.Println("已设置PowerShell执行策略为RemoteSigned")
	}
	return nil
}

// 初始化PowerShell通用代码块
// 初始化PowerShell通用代码块 - 使用字符串拼接来处理反引号
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
    function Await {
        param(
            [object]$WinRtTask,
            [Type]$ResultType
        )
        
        $asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | 
            Where-Object { 
                $_.Name -eq 'AsTask' -and 
                $_.GetParameters().Count -eq 1 -and 
                $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' 
            })[0]
        
        $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
        $netTask = $asTask.Invoke($null, @($WinRtTask))
        $netTask.Wait(-1) | Out-Null
        return $netTask.Result
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
	manager := &Win11HotspotManager{
		debug:             debug,
		commonCode:        psCommonCode,
		policyInitialized: false,
	}

	// 尝试初始化执行策略，但不阻止创建实例
	if err := manager.setExecutionPolicy(); err != nil && debug {
		fmt.Printf("初始化PowerShell执行策略警告: %v\n", err)
	}

	return manager
}

// GetStatus 获取热点状态
func (m *Win11HotspotManager) GetStatus() (models.HotspotStatus, error) {
	// 确保PowerShell执行策略已设置
	if err := m.setExecutionPolicy(); err != nil {
		return models.HotspotStatus{}, fmt.Errorf("获取热点状态前设置PowerShell执行策略失败: %v", err)
	}

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
        Authentication = $config.Authentication
        Encryption = $config.Encryption
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
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-Command", psScript)
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
		Success:        result.Success,
		Error:          result.Error,
		Enabled:        result.Enabled,
		SSID:           result.SSID,
		MaxClientCount: result.MaxClientCount,
		Authentication: result.Authentication,
		Encryption:     result.Encryption,
		ClientsCount:   result.ClientsCount,
	}, nil
}

// Configure 配置热点
func (m *Win11HotspotManager) Configure(config models.HotspotConfig) error {
	// 确保PowerShell执行策略已设置
	if err := m.setExecutionPolicy(); err != nil {
		return fmt.Errorf("配置热点前设置PowerShell执行策略失败: %v", err)
	}

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
    
    # Create new configuration
    $config = New-Object Windows.Networking.NetworkOperators.NetworkOperatorTetheringAccessPointConfiguration
    $config.Ssid = "%s"
    $config.Passphrase = "%s"
    
    # Configure the access point
    $operation = $tetheringManager.ConfigureAccessPointAsync($config)
    #$result = Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
    
    if ($%v) {
        $operation = $tetheringManager.StartTetheringAsync()
        Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
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
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-Command", psScript)
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
	// 确保PowerShell执行策略已设置
	if err := m.setExecutionPolicy(); err != nil {
		return fmt.Errorf("设置热点状态前设置PowerShell执行策略失败: %v", err)
	}

	// 构建PowerShell脚本内容
	action := "StartTetheringAsync"
	if !enable {
		action = "StopTetheringAsync"
	}

	psScript := fmt.Sprintf(`
%s
try {
    $tetheringManager = Get-TetheringManager
    
    # %s the hotspot
    $operation = $tetheringManager.%s()
    $result = Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
    
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
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-Command", psScript)
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
