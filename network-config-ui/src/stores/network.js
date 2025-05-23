import { defineStore } from 'pinia'
import { networkApi } from '../utils/api'

const MAX_LOGS = 1000 // 最大日志数量

export const useNetworkStore = defineStore('network', {
  state: () => ({
    interfaces: [],
    currentInterface: null,
    hotspots: [],
    debugLogs: [],
    logFilter: 'all', // 'all', 'error', 'success', 'info'
    debugMode: localStorage.getItem('debugMode') === 'true' || false,
    // 热点相关状态
    hotspotStatus: null,
    hotspotConfig: {
      ssid: '',
      password: ''
    }
  }),
  getters: {
    filteredLogs: (state) => {
      if (state.logFilter === 'all') {
        return state.debugLogs
      }
      return state.debugLogs.filter(log => log.type === state.logFilter)
    }
  },
  actions: {
    addDebugLog(message, content = '', type = 'info') {
      // 添加新日志
      this.debugLogs.push({
        timestamp: new Date().toISOString(),
        message,
        content: typeof content === 'string' ? content : JSON.stringify(content, null, 2),
        type
      })
      
      // 如果超过最大数量，删除最早的日志
      if (this.debugLogs.length > MAX_LOGS) {
        this.debugLogs = this.debugLogs.slice(-MAX_LOGS)
      }
    },

    async fetchHotspots(interfaceName) {
      try {
        this.hotspots = await networkApi.getWiFiHotspots(interfaceName)
        this.addDebugLog(
          `获取WIFI热点成功: ${interfaceName}`,
          this.hotspots,
          'success'
        )
      } catch (error) {
        this.addDebugLog(
          `获取WIFI热点失败: ${interfaceName}`,
          error.message,
          'error'
        )
        throw error
      }
    },
    clearDebugLogs() {
      this.debugLogs = []
    },
    setLogFilter(filter) {
      this.logFilter = filter
    },

    // 获取热点状态
    async fetchHotspotStatus() {
      try {
        this.hotspotStatus = await networkApi.getHotspotStatus()
        this.addDebugLog('获取热点状态成功', this.hotspotStatus, 'success')
      } catch (error) {
        this.addDebugLog('获取热点状态失败', error.message, 'error')
        throw error
      }
    },
    
    // 配置热点
    async configureHotspot(config) {
      try {
        const result = await networkApi.configureHotspot(config)
        this.hotspotConfig = config
        this.addDebugLog('配置热点成功', result, 'success')
        return result
      } catch (error) {
        this.addDebugLog('配置热点失败', error.message, 'error')
        throw error
      }
    },
    
    // 设置热点状态
    async setHotspotStatus(enabled) {
      try {
        const result = await networkApi.setHotspotStatus(enabled)
        this.hotspotStatus = { ...this.hotspotStatus, enabled }
        this.addDebugLog(`${enabled ? '启用' : '禁用'}热点成功`, result, 'success')
        return result
      } catch (error) {
        this.addDebugLog(`${enabled ? '启用' : '禁用'}热点失败`, error.message, 'error')
        throw error
      }
    }
  }
})