<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { AdminAuthProviders } from '../api/types'
import { loginGitHub, loginOIDC } from '../stores/auth'

const router = useRouter()
const providers = ref<AdminAuthProviders>({ github: false, providers: [] })
const loading = ref(true)
const error = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    providers.value = await api.providers()
  } catch (err: any) {
    error.value = err?.message || '加载登录方式失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)

function onGitHub(): void {
  if (!providers.value.github) {
    ElMessage.warning('服务端未配置 GitHub 登录')
    return
  }
  loginGitHub()
}

function onOIDC(id: number): void {
  loginOIDC(id)
}

function goHome(): void {
  void router.replace('/')
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card" shadow="always">
      <h1>Annotation Admin</h1>
      <p class="muted">多站点划线批注治理后台</p>

      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

      <div v-loading="loading" class="login-body">
        <el-button
          v-if="providers.github"
          type="primary"
          size="large"
          class="block"
          @click="onGitHub"
        >
          使用 GitHub 登录
        </el-button>

        <div v-if="providers.providers.length" class="providers">
          <div class="muted">或使用站点 OIDC 登录</div>
          <el-button
            v-for="provider in providers.providers"
            :key="provider.id"
            class="block"
            @click="onOIDC(provider.id)"
          >
            {{ provider.siteName }} · {{ provider.name }}
          </el-button>
        </div>

        <el-empty
          v-if="!loading && !error && !providers.github && providers.providers.length === 0"
          description="服务端尚未配置任何登录方式"
        />

        <el-button v-if="providers.github" text class="block" @click="goHome">返回首页</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--el-bg-color-page);
}
.login-card {
  width: min(420px, 100%);
}
.login-card h1 {
  margin: 0 0 4px;
  font-size: 22px;
}
.login-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 18px;
}
.providers {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}
.block {
  width: 100%;
}
</style>
