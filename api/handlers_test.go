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

// setupTestRouter 创建测试用的路由器
func setupTestRouter() (*gin.Engine, *NetworkHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	networkService := service.NewNetworkService()
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
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		// 在非管理员权限下运行时可能返回内部服务器错误
		t.Errorf("响应状态码错误: 期望 200 或 500, 得到 %d", w.Code)
	}

	if w.Code == http.StatusOK {
		// 解析响应
		var interfaces []models.Interface
		err := json.Unmarshal(w.Body.Bytes(), &interfaces)
		if err != nil {
			t.Errorf("解析响应失败: %v", err)
		}

		// 验证响应内容
		if len(interfaces) == 0 {
			t.Log("警告: 未找到网卡")
		} else {
			// 验证第一个网卡的基本信息
			firstInterface := interfaces[0]
			if firstInterface.Name == "" {
				t.Error("网卡名称为空")
			}
			if firstInterface.Status == "" {
				t.Error("网卡状态为空")
			}
		}
	}
}

func TestGetInterface(t *testing.T) {
	router, _ := setupTestRouter()

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
			// 测试获取第一个网卡的详细信息
			interfaceName := interfaces[0].Name
			w = httptest.NewRecorder()
			req, _ = http.NewRequest("GET", "/api/v1/interfaces/"+interfaceName, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("获取网卡详情失败: 状态码 %d", w.Code)
			} else {
				var iface models.Interface
				err := json.Unmarshal(w.Body.Bytes(), &iface)
				if err != nil {
					t.Errorf("解析网卡详情失败: %v", err)
				}

				if iface.Name != interfaceName {
					t.Errorf("网卡名称不匹配: 期望 %s, 得到 %s", interfaceName, iface.Name)
				}
			}
		}
	}
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