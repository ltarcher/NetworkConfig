import { defineStore } from 'pinia'

const MAX_LOGS = 1000 // 最大日志数量

export const useNetworkStore = defineStore('network', {
  state: () => ({
    interfaces: [],
    currentInterface: null,
    hotspots: [],
    debugLogs: [],
    logFilter: 'all', // 'all', 'error', 'success', 'info'
    debugMode: localStorage.getItem('debugMode') === 'true' || false
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
    }
  }
})