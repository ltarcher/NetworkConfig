import { defineStore } from 'pinia'

const MAX_LOGS = 1000 // 最大日志数量

export const useNetworkStore = defineStore('network', {
  state: () => ({
    interfaces: [],
    currentInterface: null,
    debugLogs: [],
    logFilter: 'all' // 'all', 'error', 'success', 'info'
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
    addDebugLog(message, type = 'info') {
      // 添加新日志
      this.debugLogs.push({
        timestamp: new Date().toISOString(),
        message,
        type
      })
      
      // 如果超过最大数量，删除最早的日志
      if (this.debugLogs.length > MAX_LOGS) {
        this.debugLogs = this.debugLogs.slice(-MAX_LOGS)
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