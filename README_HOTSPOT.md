# Windows 11 移动热点管理工具

这个工具用于管理Windows 11的移动热点功能，解决了Windows 11中`netsh wlan start/stop hostednetwork`命令无法操作移动热点的问题。

## 快速开始

直接使用提供的批处理文件`hotspot.bat`，无需安装Go环境。

## 使用方法

### 获取热点状态

```
hotspot.bat status
```

显示当前移动热点的状态，包括是否启用、SSID、认证方式、加密方式、最大客户端数和当前连接客户端数。

### 启用热点

```
hotspot.bat enable
```

启用移动热点。

### 禁用热点

```
hotspot.bat disable
```

禁用移动热点。

### 配置热点

```
hotspot.bat configure SSID PASSWORD [enable]
```

配置移动热点的SSID和密码。可选参数`enable`表示配置后自动启用热点。

参数说明：
- `SSID`: 热点的SSID名称
- `PASSWORD`: 热点的密码（至少8个字符）
- `enable`: 配置后自动启用热点（可选）

示例：
```
hotspot.bat configure MyWiFi Password123 enable
```

## 高级用户（需要Go环境）

如果您已安装Go环境，可以使用`build_hotspot.ps1`脚本构建可执行文件：

```powershell
.\build_hotspot.ps1
```

这将在`bin`目录下生成`hotspot.exe`可执行文件，提供与批处理文件相同的功能。

## 系统要求

- Windows 11
- 支持移动热点功能的网络适配器
- 管理员权限（某些操作需要）

## 技术实现

该工具使用Windows 11的Mobile Hotspot API（Windows.Networking.NetworkOperators命名空间）来管理移动热点，完全替代了旧的`netsh wlan hostednetwork`命令。

## 常见问题

### 为什么Windows 11中netsh命令无法操作移动热点？

Windows 11中，微软改变了移动热点的实现方式，使用了新的Mobile Hotspot API，而不再支持通过netsh命令操作移动热点。

### 如何检查我的系统是否支持移动热点？

在Windows设置中，进入"网络和Internet" > "移动热点"，如果此选项可用，则表示您的系统支持移动热点功能。

### 运行时出现错误怎么办？

确保以管理员权限运行命令提示符或PowerShell，然后再执行批处理文件。