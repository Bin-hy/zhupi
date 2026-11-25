<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { LocalAdminResult, SiteAdmin } from '../api/types'
import GeneratedPasswordDialog from './GeneratedPasswordDialog.vue'
import LocalAdminFormDialog from './LocalAdminFormDialog.vue'
import { formatTime } from '../utils/format'

const props = defineProps<{ siteId: number }>()
const admins = ref<SiteAdmin[]>([])
const loading = ref(false)
const error = ref('')
const userId = ref<number | undefined>()

const openVisible = ref(false)
const openUser = ref<{ id: number; displayName: string } | null>(null)
const passwordVisible = ref(false)
const passwordValue = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const res = await api.listSiteAdmins(props.siteId)
    admins.value = res.admins
  } catch (err: any) {
    error.value = err?.message || '加载管理员失败'
  } finally {
    loading.value = false
  }
}

async function add(): Promise<void> {
  if (!userId.value || userId.value <= 0) {
    ElMessage.warning('请输入用户 ID')
    return
  }
  try {
    await api.addSiteAdmin(props.siteId, userId.value)
    ElMessage.success('已添加站点管理员')
    userId.value = undefined
    await load()
  } catch (err: any) {
    ElMessage.error(err?.message || '添加失败')
  }
}

async function remove(admin: any): Promise<void> {
  try {
    await ElMessageBox.confirm(`移除用户 ${admin.userId} 的站点管理员身份？`, '确认', {
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.removeSiteAdmin(props.siteId, admin.userId)
    ElMessage.success('已移除')
    await load()
  } catch (err: any) {
    ElMessage.error(err?.message || '移除失败')
  }
}

function openCredential(admin: any): void {
  openUser.value = {
    id: admin.userId,
    displayName: admin.displayName || `#${admin.userId}`,
  }
  openVisible.value = true
}

function onOpenSuccess(result: LocalAdminResult): void {
  ElMessage.success('已开通本地密码')
  if (result.generatedPassword) {
    passwordValue.value = result.generatedPassword
    passwordVisible.value = true
  }
  void load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-input-number v-model="userId" :min="1" placeholder="用户 ID" controls-position="right" />
      <el-button type="primary" @click="add">添加管理员</el-button>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <el-table v-loading="loading" :data="admins" border>
      <el-table-column prop="userId" label="用户 ID" width="110" />
      <el-table-column prop="displayName" label="名称" min-width="150" />
      <el-table-column label="本地账号" width="170">
        <template #default="{ row }">
          <template v-if="row.localUsername">
            <span>{{ row.localUsername }}</span>
            <el-tag v-if="row.mustChangePassword" type="danger" size="small" class="tag">
              待改密
            </el-tag>
          </template>
          <el-button v-else link type="primary" @click="openCredential(row)">
            开通本地密码
          </el-button>
        </template>
      </el-table-column>
      <el-table-column prop="grantedBy" label="授予者" width="110" />
      <el-table-column label="授予时间" width="180">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">移除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && admins.length === 0" description="还没有站点管理员" />

    <LocalAdminFormDialog
      v-model="openVisible"
      mode="open"
      :user="openUser"
      :users="[]"
      :sites="[]"
      @success="onOpenSuccess"
    />
    <GeneratedPasswordDialog v-model="passwordVisible" :password="passwordValue" />
  </div>
</template>

<style scoped>
.tag {
  margin-left: 6px;
}
</style>
