# Windows网络配置API测试文档

## 目录
1. [测试环境设置](#测试环境设置)
2. [测试类型](#测试类型)
3. [运行测试](#运行测试)
4. [测试用例说明](#测试用例说明)
5. [测试覆盖率](#测试覆盖率)
6. [故障排除](#故障排除)

## 测试环境设置

### 必要条件
1. **开发环境要求**
   - Windows 10/11 或 Windows Server 2016/2019/2022
   - Go 1.20或更高版本
   - 管理员权限
   - PowerShell 5.0或更高版本

2. **依赖安装**
   ```powershell
   # 安装依赖包
   go mod download
   ```

3. **测试工具**
   - Go测试框架（内置）
   - PowerShell测试脚本（run_tests.ps1）
   - 测试覆盖率工具（go tool cover）

## 测试类型

### 1. 单元测试
位置：各个包中的 `*_test.go` 文件

#### Models包测试 (models/network_test.go)
- 数据结构验证
- JSON序列化/反序列化
- 字段验证

#### Service包测试 (service/network_test.go)
- 网卡信息获取
- 网络配置操作
- 错误处理
- 模拟测试

#### API包测试 (api/handlers_test.go)
- HTTP处理器测试
- 请求处理
- 响应验证
- 错误处理

### 2. 集成测试
测试组件间的交互：
- Service和API层集成
- 网络配置实际效果
- 错误传播

### 3. API测试
测试REST API接口：
- 端点可用性
- 请求/响应格式
- HTTP状态码
- 错误处理

## 运行测试

### 1. 使用测试脚本
```powershell
# 以管理员权限运行PowerShell
.\run_tests.ps1
```

### 2. 手动运行测试
```powershell
# 运行所有测试
go test ./... -v

# 运行特定包的测试
go test ./models -v
go test ./service -v
go test ./api -v

# 运行特定测试函数
go test ./models -v -run TestInterfaceJSON
```

### 3. 生成覆盖率报告
```powershell
# 生成覆盖率数据
go test ./... -coverprofile=coverage/coverage.out

# 生成HTML报告
go tool cover -html=coverage/coverage.out -o coverage/coverage.html
```

## 测试用例说明

### 1. Models包测试用例

#### TestInterfaceJSON
验证网卡信息的JSON处理：
```go
func TestInterfaceJSON(t *testing.T) {
    // 测试数据结构序列化和反序列化
    // 验证字段映射
    // 检查必需字段
}
```

### 2. Service包测试用例

#### TestGetInterfaces
测试网卡列表获取：
```go
func TestGetInterfaces(t *testing.T) {
    // 获取系统网卡列表
    // 验证返回数据
    // 检查错误处理
}
```

#### TestConfigureInterface
测试网卡配置：
```go
func TestConfigureInterface(t *testing.T) {
    // 配置网卡参数
    // 验证配置结果
    // 测试错误情况
}
```

### 3. API包测试用例

#### TestGetInterfacesAPI
测试获取网卡列表API：
```go
func TestGetInterfacesAPI(t *testing.T) {
    // 发送GET请求
    // 验证响应状态
    // 检查响应数据
}
```

#### TestConfigureIPv4API
测试IPv4配置API：
```go
func TestConfigureIPv4API(t *testing.T) {
    // 发送配置请求
    // 验证响应
    // 测试错误处理
}
```

## 测试覆盖率

### 1. 覆盖率目标
- 整体代码覆盖率 > 80%
- 核心功能覆盖率 > 90%
- 错误处理覆盖率 > 85%

### 2. 覆盖率报告解读
```
coverage/coverage.html 内容说明：
- 绿色：已覆盖的代码
- 红色：未覆盖的代码
- 灰色：不需要测试的代码
```

### 3. 改进覆盖率
- 识别未覆盖的代码路径
- 添加相应的测试用例
- 关注错误处理分支

## 故障排除

### 1. 测试失败类型

#### 权限相关
```
错误：access denied
解决：以管理员权限运行测试
```

#### 网络相关
```
错误：network interface not found
解决：确保测试环境有可用网卡
```

#### 配置相关
```
错误：invalid configuration
解决：检查测试用例中的配置参数
```

### 2. 常见问题解决

#### 测试超时
```powershell
# 增加测试超时时间
go test ./... -timeout 30m
```

#### 内存问题
```powershell
# 限制测试内存使用
go test ./... -memprofile=mem.out
```

#### 并发问题
```powershell
# 串行运行测试
go test ./... -p 1
```

## 持续集成/持续部署 (CI/CD)

### 1. GitHub Actions配置
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.20'
      - run: go test ./... -v
```

### 2. 本地CI环境
```powershell
# 运行完整测试套件
.\run_tests.ps1
```

### 3. 测试报告生成
```powershell
# 生成JUnit格式报告
go test ./... -v | go-junit-report > report.xml
```

## 测试维护

### 1. 配置测试
#### 验证配置系统
```powershell
# 运行配置验证脚本
go run scripts/verify_config.go

# 测试不同的配置方式
# 1. 命令行参数
.\NetworkConfig.exe -port 9000

# 2. 环境变量
$env:NETWORK_CONFIG_PORT="9000"
.\NetworkConfig.exe

# 3. .env文件
# 创建.env文件并设置NETWORK_CONFIG_PORT=9000
.\NetworkConfig.exe
```

#### 配置测试用例
- 验证默认端口 (8080)
- 验证命令行参数优先级
- 验证.env文件配置
- 验证无效端口处理
- 验证端口冲突处理

### 2. 测试代码审查清单
- 测试覆盖率是否满足要求
- 测试用例是否清晰明确
- 是否包含足够的错误测试
- 测试数据是否合适
- 是否遵循测试最佳实践
- 是否包含配置系统测试

### 2. 测试用例维护
- 定期审查测试用例
- 更新过时的测试
- 添加新功能的测试
- 优化测试性能

### 3. 文档更新
- 保持测试文档最新
- 记录重要的测试场景
- 更新测试运行说明
- 维护故障排除指南