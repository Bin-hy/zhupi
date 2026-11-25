<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { localLogin } from '../stores/auth'

const router = useRouter()
const available = ref(false)
const loading = ref(true)
const error = ref('')
const submitting = ref(false)
const form = ref({ username: 'admin', password: '' })

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const providers = await api.providers()
    available.value = providers.local
  } catch (err: any) {
    error.value = err?.message || '加载登录方式失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function onLocalLogin(): Promise<void> {
  if (!form.value.username.trim() || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const mustChange = await localLogin(form.value.username.trim(), form.value.password)
    await router.replace(mustChange ? '/account' : '/')
  } catch (err: any) {
    error.value = err?.message || '登录失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card" shadow="always">
      <h1>朱批 · Zhupi</h1>
      <p class="muted">多站点划线批注治理后台</p>

      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

      <div v-loading="loading" class="login-body">
        <el-form v-if="available" class="local-form" @submit.prevent="onLocalLogin">
          <div class="muted">管理员账号</div>
          <el-input
            v-model="form.username"
            placeholder="用户名"
            autocomplete="username"
            size="large"
          />
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            autocomplete="current-password"
            show-password
            size="large"
          />
          <el-button
            type="primary"
            size="large"
            class="block"
            :loading="submitting"
            native-type="submit"
          >
            登录
          </el-button>
        </el-form>

        <el-empty
          v-if="!loading && !error && !available"
          description="尚未初始化本地管理员，请联系部署人员检查配置"
        />
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
.local-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.block {
  width: 100%;
}
</style>
