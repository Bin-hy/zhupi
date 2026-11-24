<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { Annotation, Comment, Queue } from '../api/types'
import { auth } from '../stores/auth'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import LoadMore from '../components/LoadMore.vue'
import SiteSelector from '../components/SiteSelector.vue'
import StatusTag from '../components/StatusTag.vue'
import { formatTime } from '../utils/format'

const route = useRoute()
const siteId = ref<number | undefined>(undefined)
const status = ref('pending')
const annotations = ref<Annotation[]>([])
const comments = ref<Comment[]>([])
const annotationCursor = ref(0)
const commentCursor = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')

const sites = computed(() => auth.sites)

const statusOptions = [
  { value: 'pending', label: '待审' },
  { value: 'approved', label: '已通过' },
  { value: 'rejected', label: '已拒绝' },
]

function initFromRoute(): void {
  const fromQuery = Number(route.query.site)
  siteId.value = sites.value.some((s) => s.id === fromQuery)
    ? fromQuery
    : sites.value[0]?.id
  const fromStatus = String(route.query.status || '')
  status.value = statusOptions.some((s) => s.value === fromStatus) ? fromStatus : 'pending'
}

async function load(reset = true): Promise<void> {
  if (!siteId.value) return
  if (reset) {
    annotations.value = []
    comments.value = []
    annotationCursor.value = 0
    commentCursor.value = 0
    loading.value = true
  } else {
    if (annotationCursor.value === 0 && commentCursor.value === 0) return
    loadingMore.value = true
  }
  error.value = ''
  try {
    const queue: Queue = await api.queue(
      siteId.value,
      status.value,
      reset ? 0 : annotationCursor.value,
      reset ? 0 : commentCursor.value,
      50,
    )
    annotations.value = reset ? queue.annotations : [...annotations.value, ...queue.annotations]
    comments.value = reset ? queue.comments : [...comments.value, ...queue.comments]
    annotationCursor.value = queue.annotationCursor
    commentCursor.value = queue.commentCursor
  } catch (err: any) {
    error.value = err?.message || '加载审核列表失败'
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function actionsFor(item: any): string[] {
  if (item.visibility === 'hidden') return ['restore']
  if (item.moderation === 'pending') return ['approve', 'reject']
  if (item.moderation === 'approved') return ['hide']
  if (item.moderation === 'rejected') return ['approve']
  return []
}

const actionLabels: Record<string, string> = {
  approve: '通过',
  reject: '拒绝',
  hide: '隐藏',
  restore: '恢复',
}

async function moderate(kind: 'annotation' | 'comment', id: number, action: string): Promise<void> {
  try {
    await ElMessageBox.confirm(`确定对 #${id} 执行「${actionLabels[action]}」？`, '确认', {
      type: action === 'hide' || action === 'reject' ? 'warning' : 'info',
    })
  } catch {
    return
  }
  try {
    if (kind === 'annotation') {
      await api.moderateAnnotation(siteId.value!, id, action)
    } else {
      await api.moderateComment(siteId.value!, id, action)
    }
    ElMessage.success('已处理')
    await load(true)
  } catch (err: any) {
    ElMessage.error(err?.message || '操作失败')
  }
}

watch([siteId, status], () => void load(true))

onMounted(() => {
  initFromRoute()
  if (siteId.value) void load(true)
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>内容审核</h1>
        <p>批注与评论的待审、通过与拒绝</p>
      </div>
    </div>

    <el-empty v-if="sites.length === 0" description="暂无可管理站点，请联系超级管理员分配" />
    <template v-else>
      <div class="toolbar">
        <SiteSelector v-model="siteId" :sites="sites" />
        <el-radio-group v-model="status">
          <el-radio-button v-for="option in statusOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </el-radio-button>
        </el-radio-group>
        <el-button :loading="loading" @click="load(true)">刷新</el-button>
      </div>

      <ErrorState v-if="error" :message="error" @retry="load(true)" />
      <template v-else>
        <div v-loading="loading">
          <h2 class="section-title">批注</h2>
          <el-table :data="annotations" border>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column label="作者" width="140">
              <template #default="{ row }">{{ row.author.name }}</template>
            </el-table-column>
            <el-table-column prop="quoteExact" label="引用" min-width="220" show-overflow-tooltip />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusTag :status="row.visibility === 'hidden' ? 'hidden' : row.moderation" />
              </template>
            </el-table-column>
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="170" fixed="right">
              <template #default="{ row }">
                <el-button
                  v-for="action in actionsFor(row)"
                  :key="action"
                  link
                  :type="action === 'reject' || action === 'hide' ? 'danger' : 'primary'"
                  @click="moderate('annotation', row.id, action)"
                >
                  {{ actionLabels[action] }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState v-if="!loading && annotations.length === 0" description="没有符合条件的批注" />

          <h2 class="section-title" style="margin-top: 24px">评论</h2>
          <el-table :data="comments" border>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column label="作者" width="140">
              <template #default="{ row }">{{ row.author.name }}</template>
            </el-table-column>
            <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
            <el-table-column prop="annotationId" label="批注 ID" width="100" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusTag :status="row.visibility === 'hidden' ? 'hidden' : row.moderation" />
              </template>
            </el-table-column>
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="170" fixed="right">
              <template #default="{ row }">
                <el-button
                  v-for="action in actionsFor(row)"
                  :key="action"
                  link
                  :type="action === 'reject' || action === 'hide' ? 'danger' : 'primary'"
                  @click="moderate('comment', row.id, action)"
                >
                  {{ actionLabels[action] }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState v-if="!loading && comments.length === 0" description="没有符合条件的评论" />
        </div>

        <LoadMore
          :has-more="annotationCursor > 0 || commentCursor > 0"
          :loading="loadingMore"
          @load="load(false)"
        />
      </template>
    </template>
  </div>
</template>
