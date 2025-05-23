import { createRouter, createWebHistory } from 'vue-router'
import NetworkConfig from '../components/NetworkConfig.vue'
import HotspotManagement from '../components/HotspotManagement.vue'

const routes = [
  {
    path: '/',
    redirect: '/network'
  },
  {
    path: '/network',
    name: 'NetworkConfig',
    component: NetworkConfig
  },
  {
    path: '/hotspot',
    name: 'HotspotManagement',
    component: HotspotManagement
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router