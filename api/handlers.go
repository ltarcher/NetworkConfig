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
		v1.POST("/interfaces/:name/connect", h.ConnectWiFi)
		v1.GET("/interfaces/:name/hotspots", h.GetWiFiHotspots)

		// 移动热点相关接口
		v1.GET("/hotspot", h.GetHotspotStatus)
		v1.POST("/hotspot", h.ConfigureHotspot)
		v1.PUT("/hotspot/status", h.SetHotspotStatus)
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

// GetHotspotStatus 获取移动热点状态
func (h *NetworkHandler) GetHotspotStatus(c *gin.Context) {
	status, err := h.networkService.GetHotspotStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ConfigureHotspot 配置移动热点
func (h *NetworkHandler) ConfigureHotspot(c *gin.Context) {
	var config models.HotspotConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 验证请求数据
	if config.SSID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "SSID不能为空",
		})
		return
	}

	if config.Password != "" && len(config.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "密码长度必须至少为8个字符",
		})
		return
	}

	err := h.networkService.ConfigureHotspot(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "移动热点配置成功",
	})
}

// SetHotspotStatus 启用或禁用移动热点
func (h *NetworkHandler) SetHotspotStatus(c *gin.Context) {
	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求数据: " + err.Error(),
		})
		return
	}

	err := h.networkService.SetHotspotStatus(request.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	status := "启用"
	if !request.Enabled {
		status = "禁用"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "移动热点" + status + "成功",
	})
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

// ConnectWiFi 连接指定WiFi热点
// GetWiFiHotspots 获取可用WiFi热点列表
func (h *NetworkHandler) GetWiFiHotspots(c *gin.Context) {
	name := c.Param("name")

	hotspots, err := h.networkService.GetWiFiHotspots(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hotspots)
}

func (h *NetworkHandler) ConnectWiFi(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.networkService.ConnectWiFi(name, req.SSID, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "WiFi连接成功"})
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
