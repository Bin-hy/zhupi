<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { auth, loadSession, mustChangePassword } from '../stores/auth'

const router = useRouter()
const form = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' })
const submitting = ref(false)
const forced = computed(() => mustChangePassword())

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
</style>
