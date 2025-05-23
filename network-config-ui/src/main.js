import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import App from './App.vue'

// 创建Vue应用实例
const app = createApp(App)

// 注册Element Plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 使用Pinia状态管理
app.use(createPinia())

// 使用Element Plus
app.use(ElementPlus, {
  size: 'default'
})

// 使用路由
import router from './router'
app.use(router)

// 挂载应用
app.mount('#app')