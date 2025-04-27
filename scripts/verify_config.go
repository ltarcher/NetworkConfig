package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// 测试配置系统
	fmt.Println("验证配置系统...")

	// 1. 测试命令行参数
	flag.String("port", "8080", "测试用参数")
	flag.Parse()
	fmt.Printf("命令行参数解析: port flag=%s\n", flag.Lookup("port").Value)

	// 2. 测试.env文件加载
	err := godotenv.Load()
	if err != nil {
		fmt.Println("未找到.env文件 (这不是错误)")
	} else {
		fmt.Printf(".env文件加载成功: NETWORK_CONFIG_PORT=%s\n", os.Getenv("NETWORK_CONFIG_PORT"))
	}

	// 3. 测试端口验证
	testPort := "8080"
	if _, err := net.LookupPort("tcp", testPort); err != nil {
		fmt.Printf("端口验证失败: %v\n", err)
	} else {
		fmt.Printf("端口验证成功: %s\n", testPort)
	}

	fmt.Println("配置系统验证完成")
}