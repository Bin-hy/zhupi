<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { BindableProvider, Identity } from '../api/types'
import { auth, loadSession, mustChangePassword } from '../stores/auth'

const router = useRouter()
const form = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' })
const submitting = ref(false)
const forced = computed(() => mustChangePassword())

const identities = ref<Identity[]>([])
const bindable = ref<{ github: boolean; providers: BindableProvider[] }>({
  github: false,
  providers: [],
})
const identitiesLoading = ref(false)
const binding = ref('')

const groupedProviders = computed(() => {
  const groups = new Map<string, BindableProvider[]>()
  for (const provider of bindable.value.providers) {
    const key = provider.siteName || `#${provider.siteId}`
    const list = groups.get(key) || []
    list.push(provider)
    groups.set(key, list)
  }
  return Array.from(groups, ([site, providers]) => ({ site, providers }))
})

async function loadIdentities(): Promise<void> {
  identitiesLoading.value = true
  try {
    const res = await api.adminIdentities()
    identities.value = res.identities || []
    bindable.value = {
      github: res.bindable.github,
      providers: res.bindable.providers || [],
    }
  } catch (err: any) {
    ElMessage.error(err?.message || '加载登录身份失败')
  } finally {
    identitiesLoading.value = false
  }
}

function identityLabel(identity: Identity): string {
  if (identity.provider === 'github') return 'GitHub'
  return identity.issuer || identity.provider
}

function identityDetail(identity: Identity): string {
  return identity.handle || identity.email || identity.subject
}

async function bindGitHub(): Promise<void> {
  binding.value = 'github'
  try {
    const res = await api.bindGitHub()
    window.location.href = res.redirectUrl
  } catch (err: any) {
    ElMessage.error(err?.message || '发起绑定失败')
    binding.value = ''
  }
}

async function bindOIDC(providerId: number): Promise<void> {
  binding.value = `oidc-${providerId}`
  try {
    const res = await api.bindOIDC(providerId)
    window.location.href = res.redirectUrl
  } catch (err: any) {
    ElMessage.error(err?.message || '发起绑定失败')
    binding.value = ''
  }
}

async function unbind(identity: Identity): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `解绑 ${identityLabel(identity)}（${identityDetail(identity)}）？`,
      '确认',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await api.unbindIdentity(identity.id)
    ElMessage.success('已解绑')
    await loadIdentities()
  } catch (err: any) {
    ElMessage.error(err?.message || '解绑失败')
  }
}

function handleBindResult(): void {
  const hash = window.location.hash.replace(/^#/, '')
  if (!hash) return
  const params = new URLSearchParams(hash)
  const status = params.get('annot_bind')
  if (!status) return
  if (status === 'ok') {
    ElMessage.success('身份绑定成功')
  } else {
    ElMessage.error(params.get('message') || '身份绑定失败')
  }
  window.history.replaceState(null, '', window.location.pathname + window.location.search)
}

onMounted(async () => {
  handleBindResult()
  await loadIdentities()
})

async function onSubmit(): Promise<void> {
  if (!form.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (form.newPassword.length < 8) {
    ElMessage.warning('新密码至少需要 8 个字符')
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  submitting.value = true
  try {
    await api.changePassword(form.currentPassword, form.newPassword)
    ElMessage.success('密码已更新')
    form.currentPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    await loadSession()
    if (!mustChangePassword()) await router.replace('/')
  } catch (err: any) {
    ElMessage.error(err?.message || '修改密码失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="account-page">
    <el-card shadow="never">
      <template #header>
        <div class="header">
          <span>账号设置</span>
          <el-tag v-if="auth.me?.isSuperadmin" type="success">超级管理员</el-tag>
        </div>
      </template>

      <el-alert
        v-if="forced"
        type="warning"
        show-icon
        :closable="false"
        title="首次登录必须修改密码"
        description="修改成功后才能访问其他管理功能。"
        class="alert"
      />

      <el-descriptions :column="1" border class="info">
        <el-descriptions-item label="用户名">
          {{ auth.me?.username || '—' }}
        </el-descriptions-item>
        <el-descriptions-item label="显示名称">
          {{ auth.me?.displayName || '—' }}
        </el-descriptions-item>
      </el-descriptions>

      <el-form
        v-if="auth.me?.username"
        class="form"
        label-width="96px"
        @submit.prevent="onSubmit"
      >
        <el-form-item label="当前密码">
          <el-input
            v-model="form.currentPassword"
            type="password"
            show-password
            autocomplete="current-password"
          />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input
            v-model="form.newPassword"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="至少 8 个字符"
          />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" native-type="submit">
            保存新密码
          </el-button>
        </el-form-item>
      </el-form>

      <el-empty
        v-if="!auth.me?.username && !forced"
        description="当前账号未设置本地密码（通过 GitHub / OIDC 登录）"
      />
    </el-card>

    <el-card shadow="never" class="card">
      <template #header>
        <span>登录身份</span>
      </template>
      <div v-loading="identitiesLoading">
        <div v-if="identities.length" class="identity-list">
          <div v-for="identity in identities" :key="identity.id" class="identity-row">
            <div>
              <strong>{{ identityLabel(identity) }}</strong>
              <span class="muted"> {{ identityDetail(identity) }}</span>
            </div>
            <el-button link type="danger" @click="unbind(identity)">解绑</el-button>
          </div>
        </div>
        <el-empty v-else description="尚未绑定任何外部身份" :image-size="60" />
      </div>

      <div class="bind-section">
        <p class="muted">绑定 GitHub 或站点 OIDC 身份，仅用于身份关联，不参与后台登录。</p>
        <div class="bind-actions">
          <el-button
            v-if="bindable.github"
            :loading="binding === 'github'"
            @click="bindGitHub"
          >
            绑定 GitHub
          </el-button>
          <div v-for="group in groupedProviders" :key="group.site" class="bind-group">
            <span class="muted">{{ group.site }}</span>
            <el-button
              v-for="provider in group.providers"
              :key="provider.id"
              :loading="binding === `oidc-${provider.id}`"
              @click="bindOIDC(provider.id)"
            >
              绑定 {{ provider.name }}
            </el-button>
          </div>
        </div>
        <p v-if="!bindable.github && !bindable.providers.length" class="muted">
          当前没有可绑定的提供商
        </p>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.account-page {
  max-width: 720px;
  margin: 0 auto;
  padding: 24px;
}
.header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.alert {
  margin-bottom: 16px;
}
.info {
  margin-bottom: 20px;
}
.form {
  max-width: 480px;
}
.card {
  margin-top: 16px;
}
.identity-list {
  display: flex;
  flex-direction: column;
}
.identity-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.bind-section {
  margin-top: 16px;
}
.bind-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.bind-group {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
