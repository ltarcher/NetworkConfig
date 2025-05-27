package main

import (
	"flag"
	"fmt"
	"log"
	"networkconfig/models"
	"networkconfig/service"
	"os"
)

func main() {
	// 设置日志输出
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 定义子命令
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)

	enableCmd := flag.NewFlagSet("enable", flag.ExitOnError)

	disableCmd := flag.NewFlagSet("disable", flag.ExitOnError)

	configureCmd := flag.NewFlagSet("configure", flag.ExitOnError)
	ssid := configureCmd.String("ssid", "", "热点SSID名称")
	password := configureCmd.String("password", "", "热点密码")
	autoEnable := configureCmd.Bool("enable", false, "配置后自动启用热点")

	// 检查命令行参数
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 创建网络服务
	networkService := service.NewNetworkService(false)

	// 解析子命令
	switch os.Args[1] {
	case "status":
		statusCmd.Parse(os.Args[2:])
		status, err := networkService.GetHotspotStatus()
		if err != nil {
			log.Fatalf("获取热点状态失败: %v", err)
		}
		printHotspotStatus(status)

	case "enable":
		enableCmd.Parse(os.Args[2:])
		if err := networkService.SetHotspotStatus(true); err != nil {
			log.Fatalf("启用热点失败: %v", err)
		}
		fmt.Println("热点已成功启用")

	case "disable":
		disableCmd.Parse(os.Args[2:])
		if err := networkService.SetHotspotStatus(false); err != nil {
			log.Fatalf("禁用热点失败: %v", err)
		}
		fmt.Println("热点已成功禁用")

	case "configure":
		configureCmd.Parse(os.Args[2:])
		if *ssid == "" || *password == "" {
			fmt.Println("错误: 必须提供SSID和密码")
			configureCmd.PrintDefaults()
			os.Exit(1)
		}

		config := models.HotspotConfig{
			SSID:     *ssid,
			Password: *password,
			Enabled:  *autoEnable,
		}

		if err := networkService.ConfigureHotspot(config); err != nil {
			log.Fatalf("配置热点失败: %v", err)
		}
		fmt.Println("热点配置成功")

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("使用方法:")
	fmt.Println("  hotspot status                           - 获取热点状态")
	fmt.Println("  hotspot enable                           - 启用热点")
	fmt.Println("  hotspot disable                          - 禁用热点")
	fmt.Println("  hotspot configure -ssid NAME -password PWD [-enable] - 配置热点")
}

func printHotspotStatus(status models.HotspotStatus) {
	fmt.Println("移动热点状态:")
	fmt.Printf("  状态: %s\n", map[bool]string{true: "已启用", false: "已禁用"}[status.Enabled])
	fmt.Printf("  SSID: %s\n", status.SSID)
	fmt.Printf("  认证方式: %s\n", status.Authentication)
	fmt.Printf("  加密方式: %s\n", status.Encryption)
	fmt.Printf("  最大客户端数: %d\n", status.MaxClientCount)
	fmt.Printf("  当前连接客户端数: %d\n", status.ClientsCount)
}
