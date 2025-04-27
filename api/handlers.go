package api

import (
	"net/http"
	"networkconfig/models"
	"networkconfig/service"

	"github.com/gin-gonic/gin"
)

// NetworkHandler 处理网络配置相关的HTTP请求
type NetworkHandler struct {
	networkService *service.NetworkService
}

// NewNetworkHandler 创建新的NetworkHandler实例
func NewNetworkHandler(networkService *service.NetworkService) *NetworkHandler {
	return &NetworkHandler{
		networkService: networkService,
	}
}

// RegisterRoutes 注册路由
func (h *NetworkHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		v1.GET("/interfaces", h.GetInterfaces)
		v1.GET("/interfaces/:name", h.GetInterface)
		v1.PUT("/interfaces/:name/ipv4", h.ConfigureIPv4)
		v1.PUT("/interfaces/:name/ipv6", h.ConfigureIPv6)
	}
}

// GetInterfaces 获取所有网卡列表
func (h *NetworkHandler) GetInterfaces(c *gin.Context) {
	interfaces, err := h.networkService.GetInterfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, interfaces)
}

// GetInterface 获取指定网卡信息
func (h *NetworkHandler) GetInterface(c *gin.Context) {
	name := c.Param("name")
	iface, err := h.networkService.GetInterface(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, iface)
}

// ConfigureIPv4 配置IPv4
func (h *NetworkHandler) ConfigureIPv4(c *gin.Context) {
	name := c.Param("name")
	var config models.IPv4Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	err := h.networkService.ConfigureInterface(name, models.InterfaceConfig{
		IPv4Config: &config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// ConfigureIPv6 配置IPv6
func (h *NetworkHandler) ConfigureIPv6(c *gin.Context) {
	name := c.Param("name")
	var config models.IPv6Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	err := h.networkService.ConfigureInterface(name, models.InterfaceConfig{
		IPv6Config: &config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}