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
		v1.GET("/connectivity", h.CheckConnectivity)
		v1.GET("/interfaces/:name/hotspots", h.GetWiFiHotspots)
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
	var request struct {
		IPv4Config *models.IPv4Config `json:"ipv4_config"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	if request.IPv4Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少ipv4_config参数",
		})
		return
	}

	err := h.networkService.ConfigureInterface(name, models.InterfaceConfig{
		IPv4Config: request.IPv4Config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// CheckConnectivity 检查网络连通性
func (h *NetworkHandler) CheckConnectivity(c *gin.Context) {
	target := c.Query("target") // 可选参数，不传则使用默认值

	result, err := h.networkService.CheckConnectivity(target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfigureIPv6 配置IPv6
func (h *NetworkHandler) ConfigureIPv6(c *gin.Context) {
	name := c.Param("name")
	var request struct {
		IPv6Config *models.IPv6Config `json:"ipv6_config"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	if request.IPv6Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少ipv6_config参数",
		})
		return
	}

	err := h.networkService.ConfigureInterface(name, models.InterfaceConfig{
		IPv6Config: request.IPv6Config,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// GetWiFiHotspots 获取指定WIFI网卡的热点列表
func (h *NetworkHandler) GetWiFiHotspots(c *gin.Context) {
	name := c.Param("name")
	hotspots, err := h.networkService.GetAvailableWiFiHotspots(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, hotspots)
}