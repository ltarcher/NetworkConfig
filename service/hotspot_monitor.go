package service

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// HotspotMonitor 热点监控服务
type HotspotMonitor struct {
	networkService *NetworkService
	enabled        bool
	interval       time.Duration
	autoRecovery   bool
	stopChan       chan struct{}
	wg             sync.WaitGroup
	debug          bool
}

// NewHotspotMonitor 创建新的热点监控服务
func NewHotspotMonitor(networkService *NetworkService, debug bool) *HotspotMonitor {
	// 从环境变量读取配置
	enabled := getEnvBool("HOTSPOT_MONITOR_ENABLED", true)
	interval := getEnvInt("HOTSPOT_MONITOR_INTERVAL", 30)
	autoRecovery := getEnvBool("HOTSPOT_AUTO_RECOVERY", true)

	return &HotspotMonitor{
		networkService: networkService,
		enabled:        enabled,
		interval:       time.Duration(interval) * time.Second,
		autoRecovery:   autoRecovery,
		stopChan:       make(chan struct{}),
		debug:          debug,
	}
}

// Start 启动热点监控服务
func (m *HotspotMonitor) Start() {
	if !m.enabled {
		log.Println("热点监控服务未启用")
		return
	}

	m.wg.Add(1)
	go m.monitorLoop()
	log.Printf("热点监控服务已启动，监控间隔: %v, 自动恢复: %v", m.interval, m.autoRecovery)
}

// Stop 停止热点监控服务
func (m *HotspotMonitor) Stop() {
	if !m.enabled {
		return
	}

	close(m.stopChan)
	m.wg.Wait()
	log.Println("热点监控服务已停止")
}

// monitorLoop 监控循环
func (m *HotspotMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkHotspotStatus()
		}
	}
}

// checkHotspotStatus 检查热点状态
func (m *HotspotMonitor) checkHotspotStatus() {
	if m.debug {
		log.Println("正在检查热点状态...")
	}

	status, err := m.networkService.GetHotspotStatus()
	if err != nil {
		log.Printf("获取热点状态失败: %v", err)
		return
	}

	// 检查热点是否需要恢复
	if !status.Success || !status.Enabled {
		log.Printf("检测到热点异常 - Success: %v, Enabled: %v", status.Success, status.Enabled)

		if m.autoRecovery {
			m.recoverHotspot()
		} else {
			log.Println("自动恢复未启用，跳过恢复操作")
		}
	} else if m.debug {
		log.Println("热点状态正常")
	}
}

// recoverHotspot 恢复热点
func (m *HotspotMonitor) recoverHotspot() {
	log.Println("正在尝试恢复热点...")

	// 先尝试停止热点
	if err := m.networkService.SetHotspotStatus(false); err != nil {
		log.Printf("停止热点失败: %v", err)
		return
	}

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 重新启动热点
	if err := m.networkService.SetHotspotStatus(true); err != nil {
		log.Printf("启动热点失败: %v", err)
		return
	}

	log.Println("热点恢复完成")
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
