import axios from 'axios'

// 创建axios实例
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 180000
})

// 创建调试日志函数
const createDebugLogger = (store) => {
  return {
    info: (message) => store?.addDebugLog?.(message, 'info'),
    success: (message) => store?.addDebugLog?.(message, 'success'),
    error: (message) => store?.addDebugLog?.(message, 'error')
  }
}

// 请求拦截器
api.interceptors.request.use(
  config => {
    const logger = createDebugLogger(config.store)
    logger.info(`Request: ${config.method.toUpperCase()} ${config.url}`)
    if (config.data) {
      logger.info(`Request Body: ${JSON.stringify(config.data, null, 2)}`)
    }
    return config
  },
  error => {
    const logger = createDebugLogger(error.config?.store)
    logger.error(`Request Error: ${error.message}`)
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    const logger = createDebugLogger(response.config.store)
    logger.success(`Response: ${response.status} ${response.statusText}`)
    logger.info(`Response Data: ${JSON.stringify(response.data, null, 2)}`)
    return response
  },
  error => {
    const logger = createDebugLogger(error.config?.store)
    logger.error(`Response Error: ${error.message}`)
    if (error.response) {
      logger.error(`Error Data: ${JSON.stringify(error.response.data, null, 2)}`)
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
  },

  // 获取热点状态
  getHotspotStatus: async () => {
    try {
      console.log('Requesting hotspot status')
      const response = await api.get('/hotspot', {
        timeout: 30000,
        params: {
          _t: Date.now() // 防止缓存
        }
      })
      
      // 更详细的响应验证
      if (!response.data) {
        console.error('Empty response data')
        throw new Error('热点状态响应为空')
      }

      const {
        Success,
        Error: errorMsg,
        Enabled,
        SSID,
        ClientsCount,
        Authentication,
        Encryption,
        MaxClientCount
      } = response.data

      // 验证必要字段
      if (typeof Success !== 'boolean') {
        console.error('Invalid Success field:', Success)
        throw new Error('响应状态字段无效')
      }

      if (!Success && errorMsg) {
        throw new Error(errorMsg)
      }

      // 验证并规范化响应数据
      const normalizedResponse = {
        enabled: typeof Enabled === 'boolean' ? Enabled : false,
        ssid: SSID || '',
        clientsCount: typeof ClientsCount === 'number' ? 
          Math.max(0, Math.floor(ClientsCount)) : 0,
        authentication: Authentication || '',
        encryption: Encryption || '',
        maxClientCount: typeof MaxClientCount === 'number' ? 
          Math.max(0, Math.floor(MaxClientCount)) : 0
      }
      
      console.log('Normalized hotspot status:', normalizedResponse)
      return normalizedResponse
    } catch (error) {
      // 更详细的错误信息提取
      const serverMessage = error.response?.data?.message || error.message
      console.error('Hotspot status request failed:', {
        config: error.config,
        response: error.response,
        stack: error.stack
      })
      throw new Error(`获取热点状态失败: ${serverMessage}`)
    }
  },

  // 配置热点
  configureHotspot: async (config) => {
    try {
      console.log('Configuring hotspot with:', config)
      // 验证配置
      if (!config.ssid || !config.password) {
        throw new Error('SSID和密码不能为空')
      }
      if (config.password.length < 8) {
        throw new Error('密码长度至少需要8个字符')
      }
      
      const response = await api.post('/hotspot', config)
      return response.data
    } catch (error) {
      console.error('Failed to configure hotspot:', error)
      throw new Error(`配置热点失败: ${error.message}`)
    }
  },

  // 设置热点状态
  setHotspotStatus: async (enabled) => {
    try {
      console.log(`Setting hotspot status to: ${enabled}`)
      const response = await api.put('/hotspot/status', { enabled })
      return response.data
    } catch (error) {
      console.error('Failed to set hotspot status:', error)
      throw new Error(`${enabled ? '启用' : '禁用'}热点失败: ${error.message}`)
    }
  }
}

export default networkApi