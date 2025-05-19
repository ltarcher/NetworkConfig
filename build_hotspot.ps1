# 设置输出编码为UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# 检查Go是否已安装
$goPath = Get-Command go -ErrorAction SilentlyContinue
if (-not $goPath) {
    Write-Host "错误: 未找到Go命令。请确保已安装Go并添加到系统PATH中。"
    Write-Host "您可以从 https://golang.org/dl/ 下载并安装Go。"
    Write-Host "安装完成后，请重新打开PowerShell并运行此脚本。"
    exit 1
}

Write-Host "检测到Go版本："
go version

# 构建热点管理工具

# 设置输出目录
$outputDir = ".\bin"
if (-not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
    Write-Host "创建输出目录: $outputDir"
}

# 构建热点管理工具
Write-Host "正在构建热点管理工具..."
go build -o "$outputDir\hotspot.exe" .\cmd\hotspot\main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "构建成功! 可执行文件位于: $outputDir\hotspot.exe"
    Write-Host ""
    Write-Host "使用方法:"
    Write-Host "  hotspot.exe status                           - 获取热点状态"
    Write-Host "  hotspot.exe enable                           - 启用热点"
    Write-Host "  hotspot.exe disable                          - 禁用热点"
    Write-Host "  hotspot.exe configure -ssid NAME -password PWD [-enable] - 配置热点"
} else {
    Write-Host "构建失败，请检查错误信息"
}