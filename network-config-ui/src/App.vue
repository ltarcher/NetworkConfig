<script setup>
import { useRoute } from 'vue-router'
const $route = useRoute()
</script>

<template>
  <div class="app-container">
    <el-container>
      <el-header>
        <h1>网络配置工具</h1>
        <el-menu
          mode="horizontal"
          :default-active="$route.path"
          router
          background-color="#545c64"
          text-color="#fff"
          active-text-color="#ffd04b"
        >
          <el-menu-item index="/network">网卡管理</el-menu-item>
          <el-menu-item index="/hotspot">本地接入热点</el-menu-item>
        </el-menu>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<style>
body {
  margin: 0;
  padding: 0;
  font-family: var(--el-font-family);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background-color: var(--el-bg-color);
}

#app {
  height: 100vh;
}

.app-container {
  height: 100%;
}

.el-header {
  background-color: #545c64;
  color: #fff;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.el-header h1 {
  margin: 10px 0 0 20px;
  font-size: 24px;
}

.el-menu {
  margin-top: 10px;
}
</style>