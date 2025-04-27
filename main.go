package main

import (
	"flag"
	"log"
	"net"
	"networkconfig/api"
	"networkconfig/service"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 检查管理员权限
	if !isAdmin() {
		log.Fatal("此程序需要管理员权限运行")
	}

	// 创建服务实例
	networkService := service.NewNetworkService()
	networkHandler := api.NewNetworkHandler(networkService)

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

	// 启动服务器
	// 读取端口配置，优先级: 命令行参数 > .env > 默认值
	var port string
	flag.StringVar(&port, "port", "", "服务器监听端口")
	flag.Parse()

	// 如果没有命令行参数，尝试从.env读取
	if port == "" {
		_ = godotenv.Load() // 忽略错误，文件不存在也没关系
		port = os.Getenv("NETWORK_CONFIG_PORT")
	}

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
	if _, err := os.Open("\\\\.\\PHYSICALDRIVE0"); err == nil {
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