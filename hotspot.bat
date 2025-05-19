@echo off
setlocal enabledelayedexpansion

REM Check command line arguments
if "%1"=="" (
    echo Usage:
    echo   hotspot status                           - Get hotspot status
    echo   hotspot enable                           - Enable hotspot
    echo   hotspot disable                          - Disable hotspot
    echo   hotspot configure SSID PASSWORD [enable] - Configure hotspot
    exit /b 1
)

REM Parse command
set command=%1
shift

if "%command%"=="status" (
    powershell -ExecutionPolicy Bypass -Command "& {[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Add-Type -AssemblyName Windows.Networking; try { $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()); $enabled = $tetheringManager.TetheringOperationalState -eq 'On'; $ssid = $tetheringManager.GetCurrentAccessPointConfiguration().Ssid; $maxClients = $tetheringManager.GetCurrentAccessPointConfiguration().MaxClientCount; $auth = $tetheringManager.GetCurrentAccessPointConfiguration().Authentication; $encryption = $tetheringManager.GetCurrentAccessPointConfiguration().Encryption; $clientsCount = ($tetheringManager.GetTetheringClients() | Measure-Object).Count; Write-Host 'Mobile Hotspot Status:'; Write-Host ('  Status: ' + $(if ($enabled) { 'Enabled' } else { 'Disabled' })); Write-Host ('  SSID: ' + $ssid); Write-Host ('  Authentication: ' + $auth); Write-Host ('  Encryption: ' + $encryption); Write-Host ('  Max Clients: ' + $maxClients); Write-Host ('  Connected Clients: ' + $clientsCount); } catch { Write-Host ('Error: ' + $_.Exception.Message); exit 1 } }"
    exit /b %errorlevel%
)

if "%command%"=="enable" (
    powershell -ExecutionPolicy Bypass -Command "& {Add-Type -AssemblyName Windows.Networking; try { $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()); $tetheringManager.StartTetheringAsync().AsTask().Wait(); Write-Host '热点已成功启用'; } catch { Write-Host ('错误: ' + $_.Exception.Message); exit 1 } }"
    exit /b %errorlevel%
)

if "%command%"=="disable" (
    powershell -ExecutionPolicy Bypass -Command "& {Add-Type -AssemblyName Windows.Networking; try { $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()); $tetheringManager.StopTetheringAsync().AsTask().Wait(); Write-Host '热点已成功禁用'; } catch { Write-Host ('错误: ' + $_.Exception.Message); exit 1 } }"
    exit /b %errorlevel%
)

if "%command%"=="configure" (
    set ssid=%1
    set password=%2
    set enable=%3
    
    if "%ssid%"=="" (
        echo 错误: 必须提供SSID
        exit /b 1
    )
    
    if "%password%"=="" (
        echo 错误: 必须提供密码
        exit /b 1
    )
    
    set enableCmd=
    if "%enable%"=="enable" (
        set enableCmd=; $tetheringManager.StartTetheringAsync().AsTask().Wait(); Write-Host '热点已成功启用'
    )
    
    powershell -ExecutionPolicy Bypass -Command "& {Add-Type -AssemblyName Windows.Networking; try { $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager]::CreateFromConnectionProfile([Windows.Networking.Connectivity.NetworkInformation]::GetInternetConnectionProfile()); $config = New-Object Windows.Networking.NetworkOperators.NetworkOperatorTetheringAccessPointConfiguration; $config.Ssid = '%ssid%'; $config.Passphrase = '%password%'; $tetheringManager.ConfigureAccessPointAsync($config).AsTask().Wait(); Write-Host '热点配置成功'%enableCmd%; } catch { Write-Host ('错误: ' + $_.Exception.Message); exit 1 } }"
    exit /b %errorlevel%
)

echo 未知命令: %command%
echo 使用方法:
echo   hotspot status                           - 获取热点状态
echo   hotspot enable                           - 启用热点
echo   hotspot disable                          - 禁用热点
echo   hotspot configure SSID PASSWORD [enable] - 配置热点
exit /b 1