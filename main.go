package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"networkconfig/api"
	"networkconfig/service"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// 读取配置，优先级: 命令行参数 > .env > 默认值
	var (
		port  string
		debug bool
	)
	flag.StringVar(&port, "port", "", "服务器监听端口")
	flag.BoolVar(&debug, "debug", false, "启用调试模式(不过滤网卡)")
	flag.Parse()

	// 如果没有命令行参数，尝试从.env读取
	if port == "" || !debug {
		_ = godotenv.Load() // 忽略错误，文件不存在也没关系
		if port == "" {
			port = os.Getenv("NETWORK_CONFIG_PORT")
		}
		if !debug {
			debug = os.Getenv("NETWORK_CONFIG_DEBUG") == "true"
		}
	}

	// 检查管理员权限
	if !isAdmin() {
		log.Fatal("此程序需要管理员权限运行。请右键点击程序，选择'以管理员身份运行'。")
	}

	// 检查PowerShell执行策略
	cmd := exec.Command("powershell", "-Command", "Get-ExecutionPolicy")
	output, err := cmd.CombinedOutput()
	if err == nil {
		policy := strings.TrimSpace(string(output))
		log.Printf("当前PowerShell执行策略: %s", policy)
		if policy == "Restricted" {
			log.Println("警告: PowerShell执行策略为Restricted，可能影响热点管理功能。建议使用管理员权限运行以下命令：")
			log.Println("Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned")
		}
	}

	// 检查移动热点状态
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-File", "hotspot.ps1", "status")
	output, err = cmd.CombinedOutput()
	if err == nil {
		var status struct {
			Success bool `json:"Success"`
			Enabled bool `json:"Enabled"`
		}
		if err := json.Unmarshal(output, &status); err == nil {
			if status.Success {
				log.Printf("当前热点状态: %v", map[bool]string{true: "已启用", false: "已禁用"}[status.Enabled])
			} else {
				log.Printf("获取热点状态失败")
			}
		} else {
			log.Printf("解析热点状态失败: %v", err)
		}
	} else {
		log.Printf("检查热点状态失败: %v", err)
	}

	// 创建服务实例
	networkService := service.NewNetworkService(debug)
	networkHandler := api.NewNetworkHandler(networkService)

	if debug {
		log.Println("警告: 调试模式已启用，网卡列表将不过滤")
	}

	// 设置gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	router := gin.Default()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// 注册路由
	networkHandler.RegisterRoutes(router)

	// 添加健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// 设置默认值
	if port == "" {
		port = "8080"
	}

	// 验证端口格式
	if _, err := net.LookupPort("tcp", port); err != nil {
		log.Fatalf("无效的端口号: %s", port)
	}

	log.Printf("服务器启动在 http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}

// isAdmin 检查当前用户是否具有管理员权限
func isAdmin() bool {
	// 在Windows中，检查当前进程是否具有管理员权限
	// 首先检查物理驱动器访问权限
	if _, err := os.Open("\\\\.\\PHYSICALDRIVE0"); err == nil {
		return true
	}

	// 检查netsh命令权限
	cmd := exec.Command("netsh", "wlan", "show", "hostednetwork")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// corsMiddleware 处理跨域请求
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}