<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { Site } from '../api/types'
import { auth, isSuperadmin, loadSession } from '../stores/auth'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import LoadMore from '../components/LoadMore.vue'
import SiteFormDialog from '../components/SiteFormDialog.vue'
import StatusTag from '../components/StatusTag.vue'
import { formatTime, shortKey } from '../utils/format'

const router = useRouter()
const sites = ref<Site[]>([])
const nextCursor = ref(0)
const loading = ref(false)
const error = ref('')
const dialog = ref(false)
const editing = ref<Site | null>(null)

const canManage = computed(() => isSuperadmin())

async function load(reset = true): Promise<void> {
  if (reset) {
    sites.value = []
    nextCursor.value = 0
  }
  loading.value = true
  error.value = ''
  try {
    if (canManage.value) {
      const res = await api.listSites(reset ? 0 : nextCursor.value)
      sites.value = reset ? res.sites : [...sites.value, ...res.sites]
      nextCursor.value = res.nextCursor
    } else {
      sites.value = [...auth.sites]
      nextCursor.value = 0
    }
  } catch (err: any) {
    error.value = err?.message || '加载站点失败'
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editing.value = null
  dialog.value = true
}

function openEdit(site: any): void {
  editing.value = site
  dialog.value = true
}

async function onSaved(): Promise<void> {
  await load(true)
  try {
    await loadSession()
  } catch {
    /* ignore */
  }
}

async function rotate(site: any): Promise<void> {
  try {
    await ElMessageBox.confirm(`轮换「${site.name}」的 siteKey？旧 key 将立即失效。`, '确认', {
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    const result = await api.rotateSiteKey(site.id)
    await ElMessageBox.alert(result.siteKey, '新的 siteKey', { confirmButtonText: '知道了' })
    await load(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '轮换失败')
  }
}

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

onMounted(() => void load(true))
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>站点</h1>
        <p>siteKey、来源白名单与审核模式</p>
      </div>
      <div class="actions">
        <el-button :loading="loading" @click="load(true)">刷新</el-button>
        <el-button v-if="canManage" type="primary" @click="openCreate">创建站点</el-button>
      </div>
    </div>

    <ErrorState v-if="error" :message="error" @retry="load(true)" />
    <template v-else>
      <el-table v-loading="loading" :data="sites" border>
        <el-table-column prop="name" label="名称" min-width="130" />
        <el-table-column prop="displayName" label="展示名" min-width="130" />
        <el-table-column label="siteKey" min-width="200">
          <template #default="{ row }">
            <span class="mono break">{{ shortKey(row.key) }}</span>
            <el-button link type="primary" @click="copy(row.key)">复制</el-button>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="审核模式" width="110">
          <template #default="{ row }"><StatusTag :status="row.moderationMode" /></template>
        </el-table-column>
        <el-table-column label="来源" min-width="180">
          <template #default="{ row }">
            <el-tag v-for="origin in row.origins" :key="origin" size="small" class="origin-tag">
              {{ origin }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="router.push(`/sites/${row.id}`)">详情</el-button>
            <el-button v-if="canManage" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="canManage" link type="warning" @click="rotate(row)">轮换 Key</el-button>
          </template>
        </el-table-column>
      </el-table>

      <EmptyState v-if="!loading && sites.length === 0" description="还没有站点" />
      <LoadMore :has-more="nextCursor > 0" :loading="loading" @load="load(false)" />
    </template>

    <SiteFormDialog v-model="dialog" :site="editing" @saved="onSaved" />
  </div>
</template>

<style scoped>
.origin-tag {
  margin: 2px 4px 2px 0;
}
</style>
