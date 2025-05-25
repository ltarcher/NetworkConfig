package main

import (
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
	"golang.org/x/sys/windows"
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

	// 获取监听地址，优先级: 环境变量 > 默认值
	host := os.Getenv("NETWORK_CONFIG_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	// 验证端口格式
	if _, err := net.LookupPort("tcp", port); err != nil {
		log.Fatalf("无效的端口号: %s", port)
	}

	// 验证主机地址格式
	if ip := net.ParseIP(host); ip == nil {
		log.Fatalf("无效的监听地址: %s", host)
	}

	listenAddr := net.JoinHostPort(host, port)
	log.Printf("服务器启动在 http://%s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}

// isAdmin 检查当前用户是否具有管理员权限
func isAdmin() bool {
	// 使用windows包提供的API检查管理员权限
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		log.Printf("初始化SID失败: %v", err)
		// 回退到物理驱动器检查
		if _, err := os.Open("\\\\.\\PHYSICALDRIVE0"); err == nil {
			return true
		}
		return false
	}
	defer windows.FreeSid(sid)

	// 检查当前进程令牌
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		log.Printf("检查令牌成员关系失败: %v", err)
		// 回退到物理驱动器检查
		if _, err := os.Open("\\\\.\\PHYSICALDRIVE0"); err == nil {
			return true
		}
		return false
	}

	return member
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
