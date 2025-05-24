<script setup>
import { ref, onMounted } from 'vue'
import { useNetworkStore } from '../stores/network'
import { ElMessage, ElLoading } from 'element-plus'

const store = useNetworkStore()
const hotspotFormRef = ref(null)
const hotspotConfig = ref({
  ssid: '',
  password: ''
})
const configuringHotspot = ref(false)
const togglingHotspot = ref(false)
const fetchingStatus = ref(false)

// 热点表单验证规则
const hotspotRules = {
  ssid: [
    { required: true, message: '请输入热点名称', trigger: 'blur' },
    { min: 1, max: 32, message: '长度在1到32个字符之间', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码长度至少8个字符', trigger: 'blur' }
  ]
}

// 初始化热点状态
onMounted(async () => {
  const loading = ElLoading.service({
    lock: true,
    text: '正在获取热点状态...',
    background: 'rgba(0, 0, 0, 0.7)'
  })
  
  try {
    await store.fetchHotspotStatus()
    if (store.hotspotStatus?.ssid) {
      hotspotConfig.value.ssid = store.hotspotStatus.ssid
    }
  } catch (error) {
    ElMessage.error('获取热点状态失败: ' + error.message)
  } finally {
    loading.close()
  }
})

// 配置热点
const configureHotspot = async () => {
  try {
    configuringHotspot.value = true
    await hotspotFormRef.value.validate()
    await store.configureHotspot(hotspotConfig.value)
    ElMessage.success('热点配置成功')
  } catch (error) {
    ElMessage.error('配置热点失败: ' + error.message)
  } finally {
    configuringHotspot.value = false
  }
}

// 切换热点状态
const toggleHotspot = async () => {
  try {
    togglingHotspot.value = true
    const enabled = !store.hotspotStatus?.enabled
    await store.setHotspotStatus(enabled)
    ElMessage.success(enabled ? '热点已启用' : '热点已禁用')
  } catch (error) {
    ElMessage.error('操作热点状态失败: ' + error.message)
  } finally {
    togglingHotspot.value = false
  }
}
</script>

<template>
  <div class="hotspot-management">
    <el-card class="hotspot-card">
      <template #header>
        <div class="card-header">
          <span>本地接入热点管理</span>
        </div>
      </template>
      
      <el-descriptions v-if="store.hotspotStatus" :column="2" border>
        <el-descriptions-item label="当前状态">
          <el-tag :type="store.hotspotStatus.enabled ? 'success' : 'danger'">
            {{ store.hotspotStatus.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="热点名称">
          {{ store.hotspotStatus.ssid || '未配置' }}
        </el-descriptions-item>
        
        <el-descriptions-item label="认证方式">
          {{ store.hotspotStatus.authentication || 'WPA2-Personal' }}
        </el-descriptions-item>
        <el-descriptions-item label="加密方式">
          {{ store.hotspotStatus.encryption || 'AES' }}
        </el-descriptions-item>
        
        <el-descriptions-item label="最大客户端数">
          {{ store.hotspotStatus.maxClients || 8 }}
        </el-descriptions-item>
        <el-descriptions-item label="当前客户端数">
          <el-tag :type="store.hotspotStatus.clientsCount > 0 ? 'success' : 'info'">
            {{ store.hotspotStatus.clientsCount || 0 }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
      
      <el-form 
        ref="hotspotFormRef"
        :model="hotspotConfig" 
        :rules="hotspotRules"
        label-width="100px"
        class="hotspot-form"
      >
        <el-form-item label="热点名称" prop="ssid">
          <el-input
            v-model="hotspotConfig.ssid"
            placeholder="请输入热点名称"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="hotspotConfig.password"
            type="password"
            placeholder="至少8个字符"
            show-password
            clearable
          />
        </el-form-item>

        <!-- Windows API does not support MaxClients configuration -->
        
        <el-form-item>
          <el-button 
            type="primary" 
            @click="configureHotspot"
            :loading="configuringHotspot"
          >
            保存配置
          </el-button>
          <el-button 
            :type="store.hotspotStatus?.enabled ? 'danger' : 'success'"
            @click="toggleHotspot"
            :disabled="!hotspotConfig.ssid"
            :loading="togglingHotspot"
          >
            {{ store.hotspotStatus?.enabled ? '禁用热点' : '启用热点' }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.hotspot-management {
  padding: 20px;
}

.hotspot-card {
  max-width: 800px;
  margin: 0 auto;
}

.card-header {
  font-size: 18px;
  font-weight: bold;
}

.hotspot-form {
  margin-top: 20px;
}
</style>