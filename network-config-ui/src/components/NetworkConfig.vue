<template>
  <div class="network-config">
    <!-- 顶部标题 -->
    <el-header class="header">
      <h2>网络配置</h2>
      <div class="debug-toggle">
        <el-tooltip content="调试模式" placement="bottom">
          <el-switch
            v-model="debugMode"
            @change="toggleDebugMode"
            active-text="调试"
            inactive-text=""
          />
        </el-tooltip>
      </div>
    </el-header>

    <!-- 主体内容区域 -->
    <el-container class="main-container">
      <!-- 左侧接口列表 -->
      <el-aside width="250px" class="aside">
        <el-card class="interface-list">
          <template #header>
            <div class="card-header">
              <span>网络接口列表</span>
              <el-button type="primary" size="small" @click="refreshInterfaces">
                刷新
              </el-button>
            </div>
          </template>
          <el-menu
            :default-active="currentInterface?.name"
            @select="handleInterfaceSelect"
          >
            <el-menu-item
              v-for="iface in interfaces"
              :key="iface.name"
              :index="iface.name"
            >
              <el-icon><Connection /></el-icon>
              <span>{{ iface.name }}</span>
              <el-tag
                size="small"
                :type="iface.status === 'up' ? 'success' : 'danger'"
                class="status-tag"
              >
                {{ iface.status }}
              </el-tag>
              <span 
                v-if="iface.hardware?.adapter_type === 'wireless' && iface.connected_ssid"
                class="ssid-tag"
              >
                {{ iface.connected_ssid }}
              </span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-aside>

      <!-- 右侧配置面板 -->
      <el-main class="main">
        <el-card v-if="currentInterface" class="config-form">
          <template #header>
            <div class="card-header">
              <span>{{ currentInterface.name }} 配置</span>
            </div>
          </template>
          
          <!-- 接口基本信息展示 -->
          <el-descriptions :column="2" border class="interface-info">
            <el-descriptions-item label="MAC地址">
              {{ currentInterface.hardware?.mac_address || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="驱动名称">
              {{ currentInterface.driver?.name || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="DHCP状态">
              <el-tag :type="currentInterface.dhcp_enabled ? 'success' : 'info'">
                {{ currentInterface.dhcp_enabled ? '启用' : '禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="网卡类型">
              <el-tag :type="currentInterface.hardware?.adapter_type === 'wireless' ? 'warning' : ''">
                {{ currentInterface.hardware?.adapter_type === 'wireless' ? '无线网卡' : '有线网卡' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="连接热点" v-if="currentInterface.hardware?.adapter_type === 'wireless'">
              <el-tag :type="currentInterface.connected_ssid ? 'success' : 'info'">
                {{ currentInterface.connected_ssid || '未连接' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>

          <!-- 无线热点管理区域 -->
          <div v-if="currentInterface.hardware?.adapter_type === 'wireless'" class="wifi-management">
            <el-card class="wifi-list">
              <template #header>
                <div class="card-header">
                  <span>可用WiFi热点</span>
                  <el-button 
                    type="primary" 
                    size="small" 
                    @click="refreshWifiList"
                    :loading="wifiLoading">
                    刷新热点
                  </el-button>
                </div>
              </template>
              
              <el-table
                :data="wifiList"
                style="width: 100%"
                @row-click="handleWifiSelect"
                highlight-current-row>
                <el-table-column
                  prop="ssid"
                  label="热点名称"
                  width="180">
                </el-table-column>
                <el-table-column
                  prop="signal"
                  label="信号强度"
                  width="100">
                  <template #default="{row}">
                    <el-rate
                      v-model="row.signal"
                      disabled
                      :max="4"
                      :colors="['#99A9BF', '#F7BA2A', '#FF9900']">
                    </el-rate>
                  </template>
                </el-table-column>
                <el-table-column
                  prop="security"
                  label="加密类型"
                  width="120">
                </el-table-column>
              </el-table>
            </el-card>

            <!-- WiFi连接表单 -->
            <el-card class="wifi-connect" v-if="selectedWifi">
              <template #header>
                <div class="card-header">
                  <span>连接至 {{ selectedWifi.ssid }}</span>
                </div>
              </template>
              
              <el-form
                ref="wifiFormRef"
                :model="wifiForm"
                :rules="wifiRules"
                label-width="80px">
                <el-form-item label="密码" v-if="selectedWifi.security !== 'Open'">
                  <el-input
                    ref="passwordInput"
                    v-model="wifiForm.password"
                    type="password"
                    placeholder="请输入WiFi密码"
                    show-password
                    :validate-event="false"
                    @compositionstart="compositionStart"
                    @compositionend="compositionEnd">
                  </el-input>
                </el-form-item>
                
                <el-form-item>
                  <el-button
                    type="primary"
                    @click="connectWifi"
                    :loading="connecting">
                    连接
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
          
          <el-form
            ref="formRef"
            :model="ipv4Form"
            :rules="formRules"
            label-width="100px"
          >
            <el-form-item>
              <el-checkbox v-model="currentInterface.dhcp_enabled">自动获取IP和子网掩码</el-checkbox>
            </el-form-item>
            <el-form-item>
              <el-checkbox v-model="currentInterface.dns_auto">自动获取DNS</el-checkbox>
            </el-form-item>
            
            <el-form-item label="IP地址" prop="ip">
              <el-input 
                v-model="ipv4Form.ip" 
                placeholder="请输入IP地址"
                :disabled="currentInterface.dhcp_enabled" />
            </el-form-item>
            
            <el-form-item label="子网掩码" prop="mask">
              <el-input 
                v-model="ipv4Form.mask" 
                placeholder="请输入子网掩码"
                :disabled="currentInterface.dhcp_enabled" />
            </el-form-item>
            
            <el-form-item label="网关">
              <el-input 
                v-model="ipv4Form.gateway" 
                placeholder="请输入网关地址"
                :disabled="currentInterface.dhcp_enabled" />
            </el-form-item>
            
            <el-form-item label="DNS" prop="dns">
              <el-input 
                v-model="ipv4Form.dns[0]" 
                placeholder="主DNS服务器"
                :disabled="currentInterface.dns_auto" />
              <el-input 
                v-model="ipv4Form.dns[1]" 
                placeholder="备用DNS服务器"
                :disabled="currentInterface.dns_auto"
                style="margin-top: 10px;" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleSubmit">保存配置</el-button>
              <el-button @click="resetForm">重置</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-empty v-else description="请选择一个网络接口" />
      </el-main>
    </el-container>

    <!-- 底部调试控制台 -->
    <el-footer v-show="debugMode" class="footer">
      <el-card class="debug-console">
        <template #header>
          <div class="card-header">
            <div class="console-title">
              <span>调试控制台</span>
              <el-tag size="small" type="info" class="log-count">
                {{ filteredLogs.length }} 条日志
              </el-tag>
            </div>
            <div class="console-controls">
              <el-radio-group v-model="logFilter" size="small" class="filter-group">
                <el-radio-button value="all">全部</el-radio-button>
                <el-radio-button value="info">信息</el-radio-button>
                <el-radio-button value="success">成功</el-radio-button>
                <el-radio-button value="error">错误</el-radio-button>
              </el-radio-group>
              <el-button type="primary" size="small" @click="clearLogs">
                清除日志
              </el-button>
            </div>
          </div>
        </template>
        <div class="debug-logs" ref="debugLogsRef" @scroll="handleScroll">
          <div
            v-for="(log, index) in filteredLogs"
            :key="index"
            :class="['log-item', log.type]"
          >
            <span class="timestamp">{{ new Date(log.timestamp).toLocaleTimeString() }}</span>
            <span class="message">{{ log.message }}</span>
          </div>
        </div>
      </el-card>
    </el-footer>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { useNetworkStore } from '../stores/network'
import { networkApi } from '../utils/api'
import { ElMessage, ElLoading } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'

// 状态管理
const store = useNetworkStore()
const wifiList = ref([])
const selectedWifi = ref(null)
const wifiLoading = ref(false)
const connecting = ref(false)

const passwordInput = ref(null)
const isComposing = ref(false)
// 表单数据
const formRef = ref(null)
const wifiFormRef = ref(null)
const wifiForm = ref({
  password: ''
})

// 重置表单
const resetWifiForm = () => {
  wifiForm.value.password = ''
  if (wifiFormRef.value) {
    wifiFormRef.value.resetFields()
  }
}

// 输入法事件处理
const compositionStart = () => {
  isComposing.value = true
}

const compositionEnd = () => {
  isComposing.value = false
}

// 重置密码表单
const resetPasswordForm = () => {
  wifiForm.value.password = ''
}

// 空规则对象
const wifiRules = {}

const interfaces = computed(() => {
  try {
    return store.interfaces || []
  } catch (e) {
    console.error('Error in interfaces computed:', e)
    return []
  }
})

const currentInterface = computed(() => {
  try {
    return store.currentInterface || null
  } catch (e) {
    console.error('Error in currentInterface computed:', e)
    return null
  }
})

const debugLogs = computed(() => {
  try {
    return store.debugLogs || []
  } catch (e) {
    console.error('Error in debugLogs computed:', e)
    return []
  }
})
const debugMode = computed({
  get: () => store.debugMode,
  set: (value) => {
    store.debugMode = value
    localStorage.setItem('debugMode', value)
  }
})

const toggleDebugMode = (value) => {
  store.debugMode = value
  localStorage.setItem('debugMode', value)
}

const logFilter = computed({
  get: () => store.logFilter,
  set: (value) => store.setLogFilter(value)
})

const filteredLogs = computed(() => {
  try {
    return store.filteredLogs || []
  } catch (e) {
    console.error('Error in filteredLogs computed:', e)
    return []
  }
})

// 表单数据
const ipv4Form = ref({
  ip: '',
  mask: '',
  gateway: '',
  dns: ['']
})

// 表单验证规则
const formRules = {
  ip: [
    { 
      required: !currentInterface.value?.dhcp_enabled, 
      message: '请输入IP地址', 
      trigger: 'blur' 
    },
    { 
      pattern: /^(\d{1,3}\.){3}\d{1,3}$/, 
      message: '请输入有效的IP地址', 
      trigger: 'blur' 
    }
  ],
  mask: [
    { 
      required: !currentInterface.value?.dhcp_enabled, 
      message: '请输入子网掩码', 
      trigger: 'blur' 
    },
    { 
      pattern: /^(\d{1,3}\.){3}\d{1,3}$/, 
      message: '请输入有效的子网掩码', 
      trigger: 'blur' 
    }
  ]
}

// 方法
const refreshInterfaces = async () => {
  const loading = ElLoading.service({
    lock: true,
    text: '正在刷新网卡列表...',
    background: 'rgba(0, 0, 0, 0.7)'
  })
  
  try {
    const data = await networkApi.getInterfaces()
    store.interfaces = data
    if (data.length > 0 && !currentInterface.value) {
      await handleInterfaceSelect(data[0].name)
    }
  } catch (error) {
    ElMessage.error('获取网络接口列表失败')
  } finally {
    loading.close()
  }
}

const handleInterfaceSelect = async (name) => {
  const loading = ElLoading.service({
    lock: true,
    text: '正在获取网卡信息...',
    background: 'rgba(0, 0, 0, 0.7)'
  })
  
  try {
    const data = await networkApi.getInterface(name)
    store.currentInterface = data
    // 更新表单数据
    ipv4Form.value = {
      ip: data.ipv4_config?.ip || '',
      mask: data.ipv4_config?.mask || '',
      gateway: data.ipv4_config?.gateway || '',
      dns: data.ipv4_config?.dns || ['']
    }

    // 如果是无线网卡，自动刷新热点列表
    if (data.hardware?.adapter_type === 'wireless') {
      await refreshWifiList()
    }
  } catch (error) {
    ElMessage.error('获取接口详情失败')
  } finally {
    loading.close()
  }
}

// WiFi热点管理方法
const refreshWifiList = async () => {
  if (!currentInterface.value?.hardware?.adapter_type === 'wireless') {
    ElMessage.warning('当前网卡不是无线网卡')
    return
  }
  
  wifiLoading.value = true
  wifiList.value = [] // 清空列表
  
  try {
    const hotspots = await networkApi.getWiFiHotspots(currentInterface.value.name)
    
    // 转换数据格式并计算信号强度评分
    wifiList.value = hotspots.map(wifi => ({
      ssid: wifi.ssid,
      signal: calculateSignalRating(wifi.signal_strength),
      security: wifi.security,
      bssid: wifi.bssid,
      channel: wifi.channel,
      rawSignal: wifi.signal_strength
    }))
    // 按照信号强度倒序排列
    .sort((a, b) => b.rawSignal - a.rawSignal)
    
    if (wifiList.value.length === 0) {
      ElMessage.info('未发现可用WiFi热点')
    }
  } catch (error) {
    console.error('WiFi热点刷新失败:', error)
    ElMessage.error('获取WiFi列表失败: ' + error.message)
  } finally {
    wifiLoading.value = false
  }
}

// 计算信号强度评分 (0-4)
const calculateSignalRating = (strength) => {
  if (strength >= 75) return 4
  if (strength >= 50) return 3
  if (strength >= 25) return 2
  if (strength > 0) return 1
  return 0
}

const handleWifiSelect = (wifi) => {
  selectedWifi.value = wifi
  wifiForm.value.password = ''
}

const connectWifi = async () => {
  if (!selectedWifi.value) return
  
  connecting.value = true
  try {
    await networkApi.connectToWiFi({
      interface: currentInterface.value.name,
      ssid: selectedWifi.value.ssid,
      password: wifiForm.value.password
    })
    ElMessage.success('连接成功，等待网络就绪...')
    
    // 等待5秒后刷新网卡状态
    await new Promise(resolve => setTimeout(resolve, 5000))
    
    // 刷新网卡状态
    await refreshInterfaces()
    ElMessage.success('网络已就绪')
  } catch (error) {
    ElMessage.error('连接失败: ' + error.message)
  } finally {
    connecting.value = false
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    
    console.log('表单数据:', ipv4Form.value)
    
    const config = {
      ip: currentInterface.value.dhcp_enabled ? '' : ipv4Form.value.ip,
      mask: currentInterface.value.dhcp_enabled ? '' : ipv4Form.value.mask,
      gateway: ipv4Form.value.gateway,
      dns: currentInterface.value.dns_auto ? [] : ipv4Form.value.dns.filter(dns => dns && dns.trim() !== '')
    }
    
    console.log('构造的配置:', config)
    
    const requestData = {
      ipv4_config: {
        ...config,
        dhcp: currentInterface.value.dhcp_enabled,
        dnsAuto: currentInterface.value.dns_auto
      }
    }
    
    console.log('最终请求体:', requestData)
    await networkApi.updateIPv4Config(currentInterface.value.name, requestData)
    
    ElMessage.success('配置更新成功')
    await refreshInterfaces()
  } catch (error) {
    if (error.message) {
      ElMessage.error(error.message)
    }
  }
}

const resetForm = () => {
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

const clearLogs = () => {
  store.clearDebugLogs()
}

// 调试日志相关
const debugLogsRef = ref(null)
const shouldAutoScroll = ref(true)
let isUserScrolling = false

const scrollToBottom = () => {
  try {
    if (!debugLogsRef.value || !shouldAutoScroll.value) return
    
    debugLogsRef.value.scrollTop = debugLogsRef.value.scrollHeight
  } catch (e) {
    console.error('Error in scrollToBottom:', e)
  }
}

// 监听日志变化
watch(() => debugLogs.value.length, () => {
  try {
    // 使用nextTick确保DOM更新后再滚动
    nextTick(() => {
      if (!isUserScrolling && debugLogsRef.value) {
        scrollToBottom()
      }
    })
  } catch (e) {
    console.error('Error in logs watcher:', e)
  }
})

// 处理用户手动滚动
const handleScroll = () => {
  try {
    if (!debugLogsRef.value) return
    
    const { scrollTop, scrollHeight, clientHeight } = debugLogsRef.value
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50
    
    // 更新自动滚动状态
    shouldAutoScroll.value = isAtBottom
    
    // 标记用户是否正在滚动
    if (!isAtBottom) {
      isUserScrolling = true
      setTimeout(() => {
        isUserScrolling = false
      }, 1000) // 1秒后重置滚动状态
    }
  } catch (e) {
    console.error('Error in handleScroll:', e)
  }
}

// 生命周期
onMounted(() => {
  try {
    refreshInterfaces()
    
    // 添加滚动事件监听
    if (debugLogsRef.value) {
      debugLogsRef.value.addEventListener('scroll', handleScroll)
    }
  } catch (e) {
    console.error('Error in onMounted:', e)
  }
})

onUnmounted(() => {
  try {
    // 移除滚动事件监听
    if (debugLogsRef.value) {
      debugLogsRef.value.removeEventListener('scroll', handleScroll)
    }
  } catch (e) {
    console.error('Error in onUnmounted:', e)
  }
})
</script>

<style scoped>
.network-config {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.header {
  background-color: #fff;
  border-bottom: 1px solid #dcdfe6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 60px;
  flex-shrink: 0;
}

.debug-toggle {
  display: flex;
  align-items: center;
  gap: 10px;
}

.main-container {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.aside {
  background-color: #f5f7fa;
  border-right: 1px solid #dcdfe6;
  padding: 20px;
  width: 350px;
  flex-shrink: 0;
}

.main {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.footer {
  height: 300px;
  padding: 20px;
  background-color: #f5f7fa;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.interface-list {
  height: 100%;
  overflow: auto;
}

.interface-list::-webkit-scrollbar {
  width: 6px;
}

.interface-list::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.interface-list::-webkit-scrollbar-thumb {
  background: #909399;
  border-radius: 3px;
}

.interface-list::-webkit-scrollbar-thumb:hover {
  background: #606266;
}

.interface-list :deep(.el-menu) {
  border-right: none;
  overflow-x: auto;
}

.interface-list :deep(.el-menu)::-webkit-scrollbar {
  height: 6px;
}

.interface-list :deep(.el-menu)::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.interface-list :deep(.el-menu)::-webkit-scrollbar-thumb {
  background: #909399;
  border-radius: 3px;
}

.interface-list :deep(.el-menu)::-webkit-scrollbar-thumb:hover {
  background: #606266;
}

.interface-list :deep(.el-menu-item) {
  min-width: 200px;
  white-space: nowrap;
  display: flex;
  align-items: center;
  padding-right: 20px;
}

.interface-list :deep(.el-menu-item) > * {
  flex-shrink: 0;
}

.interface-list :deep(.el-menu-item) .ssid-tag {
  flex-shrink: 1;
  min-width: 0;
}

.config-form {
  max-width: 800px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-tag {
  float: right;
  margin-top: 2px;
}

.dns-input {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}

.add-dns-btn {
  margin-top: 10px;
}

.debug-console {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.debug-console :deep(.el-card__body) {
  flex: 1;
  overflow: hidden;
  padding: 0;
}

.debug-logs {
  height: 100%;
  overflow-y: auto;
  font-family: monospace;
  font-size: 12px;
  scroll-behavior: smooth;
  padding: 10px;
  box-sizing: border-box;
}

/* 自定义滚动条样式 */
.debug-logs::-webkit-scrollbar {
  width: 6px;
}

.debug-logs::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.debug-logs::-webkit-scrollbar-thumb {
  background: #909399;
  border-radius: 3px;
}

.debug-logs::-webkit-scrollbar-thumb:hover {
  background: #606266;
}

/* 日志项样式优化 */
.log-item {
  padding: 6px 8px;
  border-bottom: 1px solid #eee;
  line-height: 1.4;
  display: flex;
  align-items: flex-start;
}

.log-item:last-child {
  border-bottom: none;
}

.log-item .timestamp {
  color: #909399;
  margin-right: 10px;
  flex-shrink: 0;
  font-size: 11px;
}

.log-item .message {
  white-space: pre-wrap;
  word-break: break-all;
  flex: 1;
}

.log-item.error {
  color: #f56c6c;
  background-color: #fef0f0;
}

.log-item.success {
  color: #67c23a;
  background-color: #f0f9eb;
}

/* 调试控制台布局优化 */
.debug-console :deep(.el-card__header) {
  padding: 10px 15px;
  border-bottom: 1px solid #e4e7ed;
}

.debug-console .card-header {
  margin: 0;
  line-height: 1.5;
}

.debug-console .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
}

.console-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.console-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-group {
  margin-right: 10px;
}

.log-count {
  margin-left: 5px;
}

.debug-console .card-header .el-button {
  padding: 5px 10px;
}

.log-item {
  padding: 4px 8px;
  border-bottom: 1px solid #eee;
}

.log-item.error {
  color: #f56c6c;
  background-color: #fef0f0;
}

.log-item.success {
  color: #67c23a;
  background-color: #f0f9eb;
}

.log-item .timestamp {
  color: #909399;
  margin-right: 10px;
}

.log-item .message {
  white-space: pre-wrap;
}

.interface-info {
  margin-bottom: 20px;
  background-color: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.interface-info :deep(.el-descriptions__body) {
  background-color: #f5f7fa;
}

.interface-info :deep(.el-descriptions__label) {
  width: 100px;
  font-weight: bold;
}

.ssid-tag {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}
</style>