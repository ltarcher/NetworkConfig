# Windows网络配置API部署指南

## 目录
1. [系统要求](#系统要求)
2. [安装步骤](#安装步骤)
3. [配置说明](#配置说明)
4. [运行服务](#运行服务)
5. [故障排除](#故障排除)
6. [安全注意事项](#安全注意事项)

## 系统要求

### 硬件要求
- CPU: 1核或更多
- 内存: 最小512MB
- 磁盘空间: 最小100MB

### 软件要求
- Windows 10/11 或 Windows Server 2016/2019/2022
- Go 1.20或更高版本
- 管理员权限

### 网络要求
- 开放端口8080（可配置）
- 网络适配器访问权限

## 安装步骤

### 1. 安装Go环境
1. 访问 [Go官方下载页面](https://golang.org/dl/)
2. 下载Windows版本的Go安装包
3. 运行安装程序，按照向导完成安装
4. 验证安装：
   ```powershell
   go version
   ```

### 2. 获取项目代码
1. 克隆或下载项目代码到本地目录
2. 进入项目目录：
   ```powershell
   cd path/to/NetworkConfig
   ```

### 3. 安装依赖
```powershell
go mod download
```

### 4. 编译项目
```powershell
go build -o NetworkConfig.exe
```

## 配置说明

### 服务配置
默认配置：
- 监听端口：8080
- 绑定地址：所有接口（0.0.0.0）

如需修改配置，编辑 main.go 中的相关参数：
```go
port := "8080"  // 修改端口号
```

### 日志配置
- 默认输出到控制台
- 可通过重定向将日志写入文件：
  ```powershell
  .\NetworkConfig.exe > network_config.log 2>&1
  ```

## 运行服务

### 1. 直接运行
1. 以管理员身份打开PowerShell
2. 导航到程序目录
3. 运行程序（可选指定端口）：
   ```powershell
   # 使用默认端口8080
   .\NetworkConfig.exe
   
   # 指定端口9000
   .\NetworkConfig.exe -port 9000
   
   # 使用.env文件配置端口
   # 创建.env文件并设置NETWORK_CONFIG_PORT=9000
   .\NetworkConfig.exe
   ```

### 2. 作为Windows服务运行
1. 创建服务配置文件 network-config.xml：
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <service>
       <id>network-config</id>
       <name>Network Configuration Service</name>
       <description>Windows Network Configuration REST API Service</description>
       <executable>path\to\NetworkConfig.exe</executable>
       <logpath>path\to\logs</logpath>
       <logmode>rotate</logmode>
   </service>
   ```

2. 使用 [NSSM](https://nssm.cc/) 安装服务：
   ```powershell
   nssm install NetworkConfig path\to\NetworkConfig.exe
   nssm set NetworkConfig AppDirectory path\to\NetworkConfig
   nssm set NetworkConfig DisplayName "Network Configuration Service"
   nssm set NetworkConfig Description "Windows Network Configuration REST API Service"
   nssm set NetworkConfig Start SERVICE_AUTO_START
   ```

3. 启动服务：
   ```powershell
   Start-Service NetworkConfig
   ```

## 故障排除

### 1. 服务无法启动
检查：
- 是否以管理员权限运行
- 端口是否被占用
- Go环境是否正确配置

### 2. 网络配置失败
检查：
- 网络适配器是否存在
- 是否有足够权限
- 配置参数是否正确

### 3. 日志查看
- 检查Windows事件查看器
- 查看应用程序日志文件
- 使用debug模式运行：
  ```powershell
  $env:GIN_MODE="debug"
  .\NetworkConfig.exe
  ```

## 安全注意事项

### 1. 访问控制
- 限制API访问范围
- 配置防火墙规则
- 使用反向代理进行保护

### 2. 权限管理
- 使用最小权限原则
- 定期审查访问日志
- 及时更新系统补丁

### 3. 网络安全
- 使用HTTPS（如需要）
- 限制IP访问范围
- 监控异常访问

## 维护指南

### 1. 日常维护
- 检查日志文件大小
- 监控服务状态
- 验证网络配置

### 2. 备份策略
- 定期备份配置
- 保存网络设置快照
- 制定恢复计划

### 3. 更新升级
- 定期检查新版本
- 测试更新后的功能
- 保持依赖包更新

## 监控和告警

### 1. 服务监控
- 使用健康检查接口
- 监控CPU和内存使用
- 跟踪API响应时间

### 2. 告警设置
- 配置服务状态告警
- 监控错误率阈值
- 设置资源使用告警

### 3. 日志分析
- 收集操作日志
- 分析错误模式
- 生成使用报告