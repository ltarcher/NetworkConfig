import axios from 'axios'
import { useNetworkStore } from '../stores/network'

// 创建axios实例
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 60000
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    const store = useNetworkStore()
    store.addDebugLog(`Request: ${config.method.toUpperCase()} ${config.url}`, 'info')
    if (config.data) {
      store.addDebugLog(`Request Body: ${JSON.stringify(config.data, null, 2)}`, 'info')
    }
    return config
  },
  error => {
    const store = useNetworkStore()
    store.addDebugLog(`Request Error: ${error.message}`, 'error')
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    const store = useNetworkStore()
    store.addDebugLog(`Response: ${response.status} ${response.statusText}`, 'success')
    store.addDebugLog(`Response Data: ${JSON.stringify(response.data, null, 2)}`, 'info')
    return response
  },
  error => {
    const store = useNetworkStore()
    store.addDebugLog(`Response Error: ${error.message}`, 'error')
    if (error.response) {
      store.addDebugLog(`Error Data: ${JSON.stringify(error.response.data, null, 2)}`, 'error')
    }
    return Promise.reject(error)
  }
)

// API方法
export const networkApi = {
  // 获取所有网络接口
  getInterfaces: async () => {
    const response = await api.get('/interfaces')
    return response.data
  },

  // 获取指定网络接口详情
  getInterface: async (name) => {
    const encodedName = encodeURIComponent(name)
    const response = await api.get(`/interfaces/${encodedName}`)
    return response.data
  },

  // 更新IPv4配置
  updateIPv4Config: async (name, config) => {
    const encodedName = encodeURIComponent(name)
    // 添加请求日志
    console.log(`Sending to /interfaces/${encodedName}/ipv4:`, config)
    const response = await api.put(`/interfaces/${encodedName}/ipv4`, config)
    return response.data
  },

  // 获取WIFI热点列表
  getWiFiHotspots: async (name) => {
    try {
      const encodedName = encodeURIComponent(name)
      console.log(`Requesting WiFi hotspots for interface: ${encodedName}`)
      
      const response = await api.get(`/interfaces/${encodedName}/hotspots`, {
        timeout: 60000 // 60秒超时
      })
      
      // 验证响应数据格式
      if (!Array.isArray(response.data)) {
        throw new Error('Invalid response format: expected array')
      }
      
      // 确保每个热点有必需字段
      const validHotspots = response.data.map(hotspot => ({
        ssid: hotspot.ssid || 'Unknown',
        signal_strength: typeof hotspot.signal_strength === 'number' ? hotspot.signal_strength : 0,
        security: hotspot.security || 'Unknown',
        bssid: hotspot.bssid || '00:00:00:00:00:00',
        channel: typeof hotspot.channel === 'number' ? hotspot.channel : 0
      }))
      
      return validHotspots
    } catch (error) {
      console.error('Failed to get WiFi hotspots:', error)
      throw new Error(`获取WiFi热点列表失败: ${error.message}`)
    }
  },

  // 连接指定WiFi热点
  connectToWiFi: async (params) => {
    const { interface: iface, ssid, password } = params
    const encodedIface = encodeURIComponent(iface)
    const encodedSsid = encodeURIComponent(ssid)
    console.log(`Connecting to WiFi: ${ssid} on interface ${iface}`)
    const response = await api.post(`/interfaces/${encodedIface}/connect`, {
      ssid: encodedSsid,
      password
    })
    return response.data
  }
}

export default networkApi