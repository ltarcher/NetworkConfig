import { defineStore } from 'pinia'
import { ref } from 'vue'
import networkApi from '../utils/api'

export const useNetworkStore = defineStore('network', () => {
  // 状态
  const hotspotStatus = ref(null)
  const error = ref(null)

  // Actions
  const fetchHotspotStatus = async () => {
    try {
      error.value = null
      const status = await networkApi.getHotspotStatus()
      hotspotStatus.value = {
        enabled: status.enabled,
        ssid: status.ssid,
        authentication: status.authentication,
        encryption: status.encryption,
        maxClientCount: status.maxClientCount,
        clientsCount: status.clientsCount
      }
    } catch (err) {
      error.value = err.message
      throw err
    }
  }

  const configureHotspot = async (config) => {
    try {
      error.value = null
      await networkApi.configureHotspot(config)
      // 配置成功后刷新状态
      await fetchHotspotStatus()
    } catch (err) {
      error.value = err.message
      throw err
    }
  }

  const setHotspotStatus = async (enabled) => {
    try {
      error.value = null
      await networkApi.setHotspotStatus(enabled)
      // 状态更改后刷新状态
      await fetchHotspotStatus()
    } catch (err) {
      error.value = err.message
      throw err
    }
  }

  return {
    // 状态
    hotspotStatus,
    error,
    // Actions
    fetchHotspotStatus,
    configureHotspot,
    setHotspotStatus
  }
})