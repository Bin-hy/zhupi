<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { LocalAdminAccount, LocalAdminResult, User } from '../api/types'
import GeneratedPasswordDialog from '../components/GeneratedPasswordDialog.vue'
import LocalAdminFormDialog from '../components/LocalAdminFormDialog.vue'
import StatusTag from '../components/StatusTag.vue'
import { auth } from '../stores/auth'
import { formatTime } from '../utils/format'

const admins = ref<LocalAdminAccount[]>([])
const users = ref<User[]>([])
const loading = ref(false)
const error = ref('')
const sites = computed(() => auth.sites)

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'open'>('create')
const dialogUser = ref<User | null>(null)

const passwordVisible = ref(false)
const passwordValue = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const res = await api.listLocalAdmins()
    admins.value = res.admins
  } catch (err: any) {
    error.value = err?.message || '加载管理员失败'
  } finally {
    loading.value = false
  }
}

async function loadUsers(): Promise<void> {
  try {
    users.value = (await api.listUsers(0, 200)).users
  } catch {
    users.value = []
  }
}

function openCreate(): void {
  dialogMode.value = 'create'
  dialogUser.value = null
  dialogVisible.value = true
}

function openExisting(user?: User): void {
  dialogMode.value = 'open'
  dialogUser.value = user ?? null
  void loadUsers()
  dialogVisible.value = true
}

function onSuccess(result: LocalAdminResult): void {
  ElMessage.success('已保存')
  if (result.generatedPassword) showPassword(result.generatedPassword)
  void load()
}

function showPassword(password: string): void {
  passwordValue.value = password
  passwordVisible.value = true
}

async function resetPassword(admin: any): Promise<void> {
  let password = ''
  try {
    const res = await ElMessageBox.prompt('留空则自动生成新密码', `重置 ${admin.username} 的密码`, {
      inputType: 'password',
      inputPlaceholder: '留空自动生成',
      confirmButtonText: '重置',
      cancelButtonText: '取消',
    })
    password = res.value || ''
  } catch {
    return
  }
  try {
    const res = await api.resetLocalPassword(admin.userId, password)
    ElMessage.success('已重置，该账号其他会话已失效')
    if (res.generatedPassword) showPassword(res.generatedPassword)
    void load()
  } catch (err: any) {
    ElMessage.error(err?.message || '重置失败')
  }
}

async function toggleSuperadmin(admin: any): Promise<void> {
  const next = !admin.isSuperadmin
  try {
    await ElMessageBox.confirm(
      next ? `授予 ${admin.username} 超级管理员？` : `撤销 ${admin.username} 的超级管理员？`,
      '确认',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await api.setSuperadmin(admin.userId, next)
    ElMessage.success('已更新')
    void load()
  } catch (err: any) {
    ElMessage.error(err?.message || '操作失败')
  }
}

function siteLabel(ids: number[]): string {
  if (!ids.length) return '—'
  return ids
    .map((id) => {
      const site = sites.value.find((item) => item.id === id)
      return site ? site.displayName || site.name : `#${id}`
    })
    .join('、')
}

onMounted(() => void load())
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>管理员账号</h1>
        <p>本地密码登录、角色与站点权限</p>
      </div>
      <div class="actions">
        <el-button @click="openExisting()">从已有用户开通</el-button>
        <el-button type="primary" @click="openCreate">新建管理员</el-button>
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>

    <el-alert
      type="info"
      show-icon
      :closable="false"
      title="后台只接受本地账号密码登录；GitHub / 站点 OIDC 仅用于账号身份绑定。"
      class="notice"
    />

    <div v-if="error" class="error">{{ error }}</div>
    <el-table v-else v-loading="loading" :data="admins" border>
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="displayName" label="显示名称" min-width="140" />
      <el-table-column label="角色" min-width="180">
        <template #default="{ row }">
          <el-tag v-if="row.isSuperadmin" type="success" size="small">超级管理员</el-tag>
          <el-tag v-if="row.siteIds.length" type="warning" size="small" class="role-tag">
            站点管理员
          </el-tag>
          <el-tag v-if="!row.isSuperadmin && !row.siteIds.length" type="info" size="small">
            无管理站点
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="可管理站点" min-width="180">
        <template #default="{ row }">{{ siteLabel(row.siteIds) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }"><StatusTag :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="待改密" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.mustChangePassword" type="danger" size="small">是</el-tag>
          <span v-else class="muted">否</span>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="resetPassword(row)">重置密码</el-button>
          <el-button link :type="row.isSuperadmin ? 'warning' : 'success'" @click="toggleSuperadmin(row)">
            {{ row.isSuperadmin ? '撤销超管' : '授予超管' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <LocalAdminFormDialog
      v-model="dialogVisible"
      :mode="dialogMode"
      :user="dialogUser"
      :users="users"
      :sites="sites"
      @success="onSuccess"
    />

    <GeneratedPasswordDialog v-model="passwordVisible" :password="passwordValue" />
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 8px;
}
.notice {
  margin-bottom: 16px;
}
.role-tag {
  margin-left: 6px;
}
.error {
  padding: 16px;
  color: var(--el-color-danger);
}
</style>
