<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { SiteAdmin } from '../api/types'
import { formatTime } from '../utils/format'

const props = defineProps<{ siteId: number }>()
const admins = ref<SiteAdmin[]>([])
const loading = ref(false)
const error = ref('')
const userId = ref<number | undefined>()

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
      <el-table-column prop="userId" label="用户 ID" width="120" />
      <el-table-column prop="grantedBy" label="授予者" width="120" />
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
  </div>
</template>
