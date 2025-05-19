# Windows 11 移动热点管理工具

这个工具用于管理Windows 11的移动热点功能，解决了Windows 11中`netsh wlan start/stop hostednetwork`命令无法操作移动热点的问题。

## 功能特点

- 支持Windows 11新的移动热点API
- 向下兼容Windows 10及更早版本
- 提供简单的命令行接口
- 支持获取热点状态、启用/禁用热点、配置热点

## 使用方法

### 构建工具

在项目根目录下运行构建脚本：

```powershell
.\build_hotspot.ps1
```

这将在`bin`目录下生成`hotspot.exe`可执行文件。

### 命令行选项

#### 获取热点状态

```
hotspot.exe status
```

显示当前移动热点的状态，包括是否启用、SSID、认证方式、加密方式、最大客户端数和当前连接客户端数。

#### 启用热点

```
hotspot.exe enable
```

启用移动热点。

#### 禁用热点

```
hotspot.exe disable
```

禁用移动热点。

#### 配置热点

```
hotspot.exe configure -ssid NAME -password PWD [-enable]
```

配置移动热点的SSID和密码。可选参数`-enable`表示配置后自动启用热点。

参数说明：
- `-ssid`: 热点的SSID名称
- `-password`: 热点的密码（至少8个字符）
- `-enable`: 配置后自动启用热点（可选）

## 技术实现

该工具使用Go语言开发，通过PowerShell命令调用Windows 11的Mobile Hotspot API（Windows.Networking.NetworkOperators命名空间）来管理移动热点。对于Windows 10及更早版本，则使用传统的`netsh wlan`命令。

## 系统要求

- Windows 11或Windows 10
- 支持移动热点功能的网络适配器
- 管理员权限（某些操作需要）