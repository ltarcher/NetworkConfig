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
  }
}

export default networkApi