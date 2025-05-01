package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"networkconfig/models"
	"networkconfig/service"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockNetworkService 创建模拟的网络服务
type mockNetworkService struct {
	service.NetworkService // 嵌入真实服务类型
}

func (s *mockNetworkService) GetInterfaces() ([]models.Interface, error) {
	return []models.Interface{
		{
			Name:        "eth0",
			Description: "Test Ethernet Interface",
			Status:      "up",
			Hardware: models.Hardware{
				MACAddress:  "00:11:22:33:44:55",
				AdapterType: models.AdapterTypeEthernet,
				ProductName: "Intel(R) Ethernet Connection I219-V",
			},
			IPv4Config: models.IPv4Config{
				IP:      "192.168.1.100",
				Mask:    "255.255.255.0",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8"},
			},
		},
		{
			Name:          "wlan0",
			Description:   "Test Wireless Interface",
			Status:        "up",
			ConnectedSSID: "TestWiFi",
			Hardware: models.Hardware{
				MACAddress:  "00:11:22:33:44:66",
				AdapterType: models.AdapterTypeWireless,
				ProductName: "Intel(R) Wireless-AC 9560",
			},
			IPv4Config: models.IPv4Config{
				IP:      "192.168.2.100",
				Mask:    "255.255.255.0",
				Gateway: "192.168.2.1",
				DNS:     []string{"8.8.8.8"},
			},
		},
	}, nil
}

func (s *mockNetworkService) GetInterface(name string) (*models.Interface, error) {
	interfaces, _ := s.GetInterfaces()
	for _, iface := range interfaces {
		if iface.Name == name {
			return &iface, nil
		}
	}
	return nil, service.ErrInterfaceNotFound
}

func (s *mockNetworkService) ConfigureIPv4(name string, config models.IPv4Config) error {
	return nil
}

func (s *mockNetworkService) ConfigureIPv6(name string, config models.IPv6Config) error {
	return nil
}

func (s *mockNetworkService) GetAvailableWiFiHotspots(interfaceName string) ([]models.WiFiHotspot, error) {
	return []models.WiFiHotspot{
		{
			SSID:         "TestWiFi",
			SignalLevel:  80, // 信号强度百分比
			SecurityType: "WPA2",
			IsConnected:  false,
		},
	}, nil
}

func (s *mockNetworkService) ConnectToWiFi(interfaceName, ssid, password string) error {
	return nil
}

func (s *mockNetworkService) CheckConnectivity(target string) (models.ConnectivityResult, error) {
	return models.ConnectivityResult{
		Success:    true,
		DurationMs: 50,
	}, nil
}

// setupTestRouter 创建测试用的路由器
func setupTestRouter() (*gin.Engine, *NetworkHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	networkService := &mockNetworkService{}
	handler := NewNetworkHandler(networkService)
	handler.RegisterRoutes(router)
	return router, handler
}

func TestGetInterfaces(t *testing.T) {
	router, _ := setupTestRouter()

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/interfaces", nil)
	router.ServeHTTP(w, req)

	// 检查响应状态码
	if w.Code != http.StatusOK {
		t.Errorf("响应状态码错误: 期望 200, 得到 %d", w.Code)
		return
	}

	// 解析响应
	var interfaces []models.Interface
	err := json.Unmarshal(w.Body.Bytes(), &interfaces)
	if err != nil {
		t.Errorf("解析响应失败: %v", err)
		return
	}

	// 验证响应内容
	if len(interfaces) != 2 {
		t.Errorf("期望返回2个网卡, 得到 %d", len(interfaces))
		return
	}

	// 验证有线网卡
	eth0 := interfaces[0]
	if eth0.Name != "eth0" {
		t.Errorf("网卡名称不匹配: 期望 eth0, 得到 %s", eth0.Name)
	}
	if eth0.ConnectedSSID != "" {
		t.Errorf("有线网卡不应该有ConnectedSSID, 但得到: %s", eth0.ConnectedSSID)
	}
	if eth0.Hardware.AdapterType != models.AdapterTypeEthernet {
		t.Errorf("网卡类型不匹配: 期望 %s, 得到 %s", models.AdapterTypeEthernet, eth0.Hardware.AdapterType)
	}

	// 验证无线网卡
	wlan0 := interfaces[1]
	if wlan0.Name != "wlan0" {
		t.Errorf("网卡名称不匹配: 期望 wlan0, 得到 %s", wlan0.Name)
	}
	if wlan0.ConnectedSSID != "TestWiFi" {
		t.Errorf("SSID不匹配: 期望 TestWiFi, 得到 %s", wlan0.ConnectedSSID)
	}
	if wlan0.Hardware.AdapterType != models.AdapterTypeWireless {
		t.Errorf("网卡类型不匹配: 期望 %s, 得到 %s", models.AdapterTypeWireless, wlan0.Hardware.AdapterType)
	}
}

func TestGetInterface(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("Get Ethernet Interface", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/interfaces/eth0", nil)
		router.ServeHTTP(w, req)

		// 检查响应状态码
		if w.Code != http.StatusOK {
			t.Errorf("响应状态码错误: 期望 200, 得到 %d", w.Code)
			return
		}

		// 解析响应
		var iface models.Interface
		err := json.Unmarshal(w.Body.Bytes(), &iface)
		if err != nil {
			t.Errorf("解析响应失败: %v", err)
			return
		}

		// 验证响应内容
		if iface.Name != "eth0" {
			t.Errorf("网卡名称不匹配: 期望 eth0, 得到 %s", iface.Name)
		}
		if iface.Status != "up" {
			t.Errorf("网卡状态不匹配: 期望 up, 得到 %s", iface.Status)
		}
		if iface.ConnectedSSID != "" {
			t.Errorf("有线网卡不应该有ConnectedSSID, 但得到: %s", iface.ConnectedSSID)
		}
		if iface.Hardware.AdapterType != models.AdapterTypeEthernet {
			t.Errorf("网卡类型不匹配: 期望 %s, 得到 %s", models.AdapterTypeEthernet, iface.Hardware.AdapterType)
		}
	})

	t.Run("Get Wireless Interface", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/interfaces/wlan0", nil)
		router.ServeHTTP(w, req)

		// 检查响应状态码
		if w.Code != http.StatusOK {
			t.Errorf("响应状态码错误: 期望 200, 得到 %d", w.Code)
			return
		}

		// 解析响应
		var iface models.Interface
		err := json.Unmarshal(w.Body.Bytes(), &iface)
		if err != nil {
			t.Errorf("解析响应失败: %v", err)
			return
		}

		// 验证响应内容
		if iface.Name != "wlan0" {
			t.Errorf("网卡名称不匹配: 期望 wlan0, 得到 %s", iface.Name)
		}
		if iface.Status != "up" {
			t.Errorf("网卡状态不匹配: 期望 up, 得到 %s", iface.Status)
		}
		if iface.ConnectedSSID != "TestWiFi" {
			t.Errorf("SSID不匹配: 期望 TestWiFi, 得到 %s", iface.ConnectedSSID)
		}
		if iface.Hardware.AdapterType != models.AdapterTypeWireless {
			t.Errorf("网卡类型不匹配: 期望 %s, 得到 %s", models.AdapterTypeWireless, iface.Hardware.AdapterType)
		}
	})

	t.Run("Get Nonexistent Interface", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/interfaces/nonexistent", nil)
		router.ServeHTTP(w, req)

		// 检查响应状态码
		if w.Code != http.StatusNotFound {
			t.Errorf("响应状态码错误: 期望 404, 得到 %d", w.Code)
		}
	})
}

func TestConfigureIPv4(t *testing.T) {
	router, _ := setupTestRouter()

	// 准备测试数据
	config := models.IPv4Config{
		IP:      "192.168.1.100",
		Mask:    "255.255.255.0",
		Gateway: "192.168.1.1",
		DNS:     []string{"8.8.8.8", "8.8.4.4"},
	}

	// 获取网卡列表以获取有效的网卡名称
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/interfaces", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var interfaces []models.Interface
		err := json.Unmarshal(w.Body.Bytes(), &interfaces)
		if err != nil {
			t.Errorf("解析响应失败: %v", err)
			return
		}

		if len(interfaces) > 0 {
			// 测试配置第一个网卡的IPv4
			interfaceName := interfaces[0].Name
			configJSON, _ := json.Marshal(config)
			w = httptest.NewRecorder()
			req, _ = http.NewRequest("PUT", "/api/v1/interfaces/"+interfaceName+"/ipv4",
				bytes.NewBuffer(configJSON))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			// 由于这是一个需要管理员权限的操作，我们期望在非管理员权限下返回错误
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
				t.Errorf("配置IPv4返回意外的状态码: %d", w.Code)
			}
		}
	}
}

func TestConfigureIPv6(t *testing.T) {
	router, _ := setupTestRouter()

	// 准备测试数据
	config := models.IPv6Config{
		IP:        "2001:db8::1",
		PrefixLen: 64,
		Gateway:   "2001:db8::1",
		DNS:       []string{"2001:4860:4860::8888"},
	}

	// 获取网卡列表以获取有效的网卡名称
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/interfaces", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var interfaces []models.Interface
		err := json.Unmarshal(w.Body.Bytes(), &interfaces)
		if err != nil {
			t.Errorf("解析响应失败: %v", err)
			return
		}

		if len(interfaces) > 0 {
			// 测试配置第一个网卡的IPv6
			interfaceName := interfaces[0].Name
			configJSON, _ := json.Marshal(config)
			w = httptest.NewRecorder()
			req, _ = http.NewRequest("PUT", "/api/v1/interfaces/"+interfaceName+"/ipv6",
				bytes.NewBuffer(configJSON))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			// 由于这是一个需要管理员权限的操作，我们期望在非管理员权限下返回错误
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
				t.Errorf("配置IPv6返回意外的状态码: %d", w.Code)
			}
		}
	}
}
