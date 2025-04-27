# 设置输出编码为UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Write-Host "开始运行测试..." -ForegroundColor Green

# 创建测试覆盖率输出目录
if (-not (Test-Path "coverage")) {
    New-Item -ItemType Directory -Path "coverage"
}

# 运行测试并生成覆盖率报告
Write-Host "`n运行所有测试并生成覆盖率报告..." -ForegroundColor Cyan
go test ./... -v -coverprofile=coverage/coverage.out

# 如果测试成功，则生成HTML覆盖率报告
if ($LASTEXITCODE -eq 0) {
    Write-Host "`n生成HTML覆盖率报告..." -ForegroundColor Cyan
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    
    # 显示总体覆盖率
    Write-Host "`n总体覆盖率统计：" -ForegroundColor Cyan
    go tool cover -func=coverage/coverage.out

    Write-Host "`n测试完成！" -ForegroundColor Green
    Write-Host "覆盖率报告已生成在 coverage/coverage.html" -ForegroundColor Green
} else {
    Write-Host "`n测试失败！" -ForegroundColor Red
}

# 等待用户按键继续
Write-Host "`n按任意键继续..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")