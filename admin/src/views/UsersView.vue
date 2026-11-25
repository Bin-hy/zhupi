<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { LocalAdminResult, User } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import GeneratedPasswordDialog from '../components/GeneratedPasswordDialog.vue'
import LoadMore from '../components/LoadMore.vue'
import LocalAdminFormDialog from '../components/LocalAdminFormDialog.vue'
import StatusTag from '../components/StatusTag.vue'
import { auth } from '../stores/auth'
import { formatTime } from '../utils/format'

const users = ref<User[]>([])
const nextCursor = ref(0)
const loading = ref(false)
const error = ref('')
const search = ref('')
const localMap = ref<Record<number, string>>({})
const sites = computed(() => auth.sites)

const openVisible = ref(false)
const openUser = ref<{ id: number; displayName: string } | null>(null)
const passwordVisible = ref(false)
const passwordValue = ref('')

const filtered = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter(
    (user) =>
      String(user.id).includes(keyword) || (user.displayName || '').toLowerCase().includes(keyword),
  )
})

async function load(reset = true): Promise<void> {
  if (reset) {
    users.value = []
    nextCursor.value = 0
  }
  loading.value = true
  error.value = ''
  try {
    const [res] = await Promise.all([api.listUsers(reset ? 0 : nextCursor.value), loadLocal()])
    users.value = reset ? res.users : [...users.value, ...res.users]
    nextCursor.value = res.nextCursor
  } catch (err: any) {
    error.value = err?.message || '加载用户失败'
  } finally {
    loading.value = false
  }
}

async function loadLocal(): Promise<void> {
  try {
    const res = await api.listLocalAdmins()
    const map: Record<number, string> = {}
    for (const admin of res.admins) map[admin.userId] = admin.username
    localMap.value = map
  } catch {
    localMap.value = {}
  }
}

function openCredential(user: any): void {
  openUser.value = { id: user.id, displayName: user.displayName || `#${user.id}` }
  openVisible.value = true
}

function onOpenSuccess(result: LocalAdminResult): void {
  ElMessage.success('已开通本地密码')
  if (result.generatedPassword) {
    passwordValue.value = result.generatedPassword
    passwordVisible.value = true
  }
  void load(true)
}

async function setBan(user: any, banned: boolean): Promise<void> {
  const message = banned
    ? `封禁用户 ${user.displayName || user.id}？其全部会话会立即失效。`
    : `解封用户 ${user.displayName || user.id}？`
  try {
    await ElMessageBox.confirm(message, '确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    if (banned) await api.banUser(user.id)
    else await api.unbanUser(user.id)
    ElMessage.success(banned ? '已封禁' : '已解封')
    await load(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '操作失败')
  }
}

onMounted(() => void load(true))
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>用户管理</h1>
        <p>全局用户与封禁状态</p>
      </div>
      <el-button :loading="loading" @click="load(true)">刷新</el-button>
    </div>

    <div class="toolbar">
      <el-input v-model="search" placeholder="搜索名称 / ID" style="width: 240px" clearable />
    </div>

    <ErrorState v-if="error" :message="error" @retry="load(true)" />
    <template v-else>
      <el-table v-loading="loading" :data="filtered" border>
        <el-table-column prop="id" label="ID" width="90" />
        <el-table-column prop="displayName" label="名称" min-width="180" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="角色" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.isSuperadmin" type="success" size="small">超管</el-tag>
            <el-tag v-else type="info" size="small">用户</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="本地账号" width="170">
          <template #default="{ row }">
            <span v-if="localMap[row.id]">{{ localMap[row.id] }}</span>
            <el-button
              v-else
              link
              type="primary"
              :disabled="row.status === 'banned'"
              @click="openCredential(row)"
            >
              开通本地密码
            </el-button>
          </template>
        </el-table-column>
        <el-table-column label="注册时间" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 'banned'"
              link
              type="danger"
              @click="setBan(row, true)"
            >
              封禁
            </el-button>
            <el-button v-else link type="primary" @click="setBan(row, false)">解封</el-button>
          </template>
        </el-table-column>
      </el-table>
      <EmptyState v-if="!loading && filtered.length === 0" description="没有用户" />
      <LoadMore :has-more="nextCursor > 0" :loading="loading" @load="load(false)" />
    </template>

    <LocalAdminFormDialog
      v-model="openVisible"
      mode="open"
      :user="openUser"
      :users="users"
      :sites="sites"
      @success="onOpenSuccess"
    />
    <GeneratedPasswordDialog v-model="passwordVisible" :password="passwordValue" />
  </div>
</template>
