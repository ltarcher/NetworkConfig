# 获取当前构建时间并格式化为YYYYMMDD-HHMMSS
$buildTime = Get-Date -Format "yyyyMMdd-HHmmss"

# 获取git commit id
$commitId = git rev-parse HEAD

# 如果获取commit id失败，使用默认值
if ($LASTEXITCODE -ne 0) {
    Write-Host "Warning: Unable to get git commit id, using 'development' as version"
    $commitId = "development"
}

# 组合构建时间和commit id
$version = "$buildTime-$commitId"

# 构建可执行文件
$buildCmd = "go build -ldflags ""-X main.version=$version"" -o release/NetworkConfig.exe"
Write-Host "Building with command: $buildCmd"
Invoke-Expression $buildCmd

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Version: $commitId"
} else {
    Write-Host "Build failed!"
    exit 1
}