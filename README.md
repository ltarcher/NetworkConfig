# Windows网络配置 REST API

一个基于Go语言开发的Windows网络配置REST API服务，提供网卡信息获取和配置功能。

## 功能特点

- 获取系统网卡列表
- 获取单个网卡详细信息（包含硬件和驱动信息）
- 配置网卡IPv4参数（IP地址、子网掩码、网关、DNS）
- 配置网卡IPv6参数（IP地址、前缀长度、网关、DNS）
- 完整的中文支持
- RESTful API设计
- 支持跨域请求
- 完整的错误处理
- 详细的日志记录
- 灵活的配置系统：
  - 命令行参数 (`-port`)
  - 环境变量/`.env`文件
  - 默认值

## 快速开始

### 前提条件

- Windows 10/11 或 Windows Server 2016/2019/2022
- Go 1.20或更高版本
- 管理员权限

### 安装

1. 克隆项目：
```bash
git clone [repository-url]
cd NetworkConfig
```

2. 安装依赖：
```bash
go mod download
```

3. 编译项目：
```bash
go build -o NetworkConfig.exe
```

### 运行

以管理员权限运行：
```bash
# 使用默认端口8080
.\NetworkConfig.exe

# 指定端口
.\NetworkConfig.exe -port 9000

# 或者通过.env文件配置
# 创建.env文件并设置NETWORK_CONFIG_PORT=9000
# 然后运行
.\NetworkConfig.exe
```

服务默认监听 http://localhost:8080

## Docker 部署

### 前提条件
- 已安装 Docker 和 Docker Compose
- 确保 Docker 服务正在运行

### 使用 Docker 运行

1. 构建镜像：
```bash
docker build -t network-config .
```

2. 运行容器：
```bash
docker run -d --name network-config -p 8080:8080 --restart unless-stopped network-config
```

3. 访问服务：
打开浏览器访问 http://localhost:8080

### 使用 Docker Compose 运行

1. 启动服务：
```bash
docker-compose up -d
```

2. 停止服务：
```bash
docker-compose down
```

3. 查看日志：
```bash
docker-compose logs -f
```

### 注意事项
- 容器内需要管理员权限，运行时添加 `--privileged` 参数
- 默认监听端口可在 docker-compose.yml 中修改
- Windows 容器需要使用 `mcr.microsoft.com/windows/nanoserver` 基础镜像

## API接口

### 获取网卡列表
```
GET /api/v1/interfaces
```

响应示例：
```json
[
  {
    "name": "以太网",
    "description": "Intel(R) Ethernet Connection",
    "status": "up",
    "ipv4_config": {
      "ip": "192.168.1.100",
      "mask": "255.255.255.0",
      "gateway": "192.168.1.1",
      "dns": ["8.8.8.8", "8.8.4.4"]
    },
    "ipv6_config": {
      "ip": "fe80::1234:5678:9abc:def0",
      "prefix_len": 64,
      "gateway": "fe80::1",
      "dns": ["2001:4860:4860::8888"]
    },
    "hardware": {
      "mac_address": "00:11:22:33:44:55",
      "manufacturer": "Intel Corporation",
      "product_name": "Intel(R) Ethernet Connection I219-V",
      "adapter_type": "Ethernet 802.3",
      "physical_media": "Ethernet",
      "speed": "1000 Mbps",
      "bus_type": "PCI",
      "pnp_device_id": "PCI\\VEN_8086&DEV_15B8"
    },
    "driver": {
      "name": "Intel(R) Ethernet Connection I219-V",
      "version": "12.18.9.23",
      "provider": "Intel",
      "date_installed": "2024-01-01",
      "status": "OK",
      "path": "C:\\Windows\\System32\\DriverStore\\FileRepository\\e1d68x64.inf_amd64_abc123\\e1d68x64.inf"
    }
  }
]
```

### 获取单个网卡信息
```
GET /api/v1/interfaces/{name}
```

### 配置网卡IPv4
```
PUT /api/v1/interfaces/{name}/ipv4
```

请求体示例：
```json
{
  "ip": "192.168.1.100",
  "mask": "255.255.255.0",
  "gateway": "192.168.1.1",
  "dns": ["8.8.8.8", "8.8.4.4"]
}
```

### 配置网卡IPv6
```
PUT /api/v1/interfaces/{name}/ipv6
```

请求体示例：
```json
{
  "ip": "2001:db8::1",
  "prefix_len": 64,
  "gateway": "2001:db8::1",
  "dns": ["2001:4860:4860::8888"]
}
```

## 项目结构

```
NetworkConfig/
├── main.go              # 主程序入口
├── api/                 # API 处理层
│   └── handlers.go      # API 处理函数
├── service/             # 业务逻辑层
│   ├── network.go       # 网络配置相关业务逻辑
│   └── encoding.go      # 编码处理相关功能
├── models/              # 数据模型
│   └── network.go       # 网络相关数据结构
└── docs/               # 文档
    ├── deployment.md   # 部署文档
    └── testing.md      # 测试文档
```

## 文档

- [部署指南](docs/deployment.md)
- [测试文档](docs/testing.md)

## 测试

运行测试：
```bash
.\run_tests.ps1
```

查看测试覆盖率报告：
```bash
coverage/coverage.html
```

## 注意事项

1. **管理员权限**
   - 程序需要管理员权限才能修改网络配置
   - 确保以管理员身份运行程序

2. **网络配置**
   - 修改网络配置前建议备份当前配置
   - 确保提供正确的网络参数
   - 配置更改可能需要几秒钟生效

3. **安全性**
   - 默认监听本地端口
   - 建议在受信任的网络环境中使用
   - 可以配置防火墙规则限制访问

4. **中文支持**
   - 支持中文网卡名称和描述
   - 正确处理中文编码
   - 日志支持中文输出

## 故障排除

1. **服务无法启动**
   - 检查是否以管理员权限运行
   - 确认端口8080未被占用
   - 验证Go环境配置

2. **网卡配置失败**
   - 确认网卡名称正确
   - 验证配置参数格式
   - 检查系统权限

3. **API访问问题**
   - 确认服务正在运行
   - 检查网络连接
   - 验证API请求格式

4. **中文显示问题**
   - 确认系统使用UTF-8编码
   - 检查PowerShell输出编码设置
   - 验证日志文件编码

5. **硬件信息获取失败**
   - 确认WMI服务正在运行
   - 检查网卡驱动是否正确安装
   - 验证系统权限设置

## 贡献

欢迎提交问题和改进建议！

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。