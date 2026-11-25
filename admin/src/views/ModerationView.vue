<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type {
  Annotation,
  AnnotationBrief,
  Comment,
  ModerationComment,
  ModerationCounts,
  Queue,
} from '../api/types'
import { auth } from '../stores/auth'
import { scheduleModerationBadgeRefresh } from '../stores/moderation'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import LoadMore from '../components/LoadMore.vue'
import SiteSelector from '../components/SiteSelector.vue'
import StatusTag from '../components/StatusTag.vue'
import { formatTime } from '../utils/format'
import { buildPageLink } from '../utils/page'

type Status = 'pending' | 'approved' | 'rejected'
type TypeFilter = 'all' | 'annotation' | 'comment'
type TableRow = Record<PropertyKey, any>
type RowLike = TableRow | Annotation | AnnotationBrief | Comment | ModerationComment

const route = useRoute()
const router = useRouter()
const siteId = ref<number | undefined>(undefined)
const status = ref<Status>('pending')
const typeFilter = ref<TypeFilter>('all')
const annotations = ref<Annotation[]>([])
const comments = ref<ModerationComment[]>([])
const counts = ref<ModerationCounts>({
  annotations: { pending: 0, approved: 0, rejected: 0 },
  comments: { pending: 0, approved: 0, rejected: 0 },
})
const annotationCursor = ref(0)
const commentCursor = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const bulkLoading = ref(false)
const error = ref('')
const searchInput = ref('')
const keyword = ref('')
const selectedAnnotations = ref<Annotation[]>([])
const selectedComments = ref<ModerationComment[]>([])
const contextVisible = ref(false)
const contextLoading = ref(false)
const contextError = ref('')
const contextAnnotation = ref<Annotation | null>(null)
const contextFocusId = ref<number | null>(null)
const contextTitle = ref('批注上下文')

let searchTimer: number | undefined
let lastQueryKey = ''

const sites = computed(() => auth.sites)
const currentSite = computed(() => sites.value.find((site) => site.id === siteId.value))

const statusOptions: Array<{ value: Status; label: string }> = [
  { value: 'pending', label: '待审' },
  { value: 'approved', label: '已通过' },
  { value: 'rejected', label: '已拒绝' },
]

const typeOptions: Array<{ value: TypeFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'annotation', label: '批注' },
  { value: 'comment', label: '评论' },
]

const actionLabels: Record<string, string> = {
  approve: '通过',
  reject: '拒绝',
  hide: '隐藏',
  restore: '恢复',
}

const statusTotals = computed<Record<Status, number>>(() => ({
  pending: counts.value.annotations.pending + counts.value.comments.pending,
  approved: counts.value.annotations.approved + counts.value.comments.approved,
  rejected: counts.value.annotations.rejected + counts.value.comments.rejected,
}))

const typeTotals = computed<Record<TypeFilter, number>>(() => ({
  all: statusTotals.value[status.value],
  annotation: counts.value.annotations[status.value],
  comment: counts.value.comments[status.value],
}))

const annotationBulkActions = computed(() => commonActions(selectedAnnotations.value))
const commentBulkActions = computed(() => commonActions(selectedComments.value))

const hasMore = computed(() => {
  if (typeFilter.value === 'annotation') return annotationCursor.value > 0
  if (typeFilter.value === 'comment') return commentCursor.value > 0
  return annotationCursor.value > 0 || commentCursor.value > 0
})

function queryKey(): string {
  return [siteId.value, status.value, typeFilter.value, keyword.value].join('|')
}

function initFromRoute(): void {
  window.clearTimeout(searchTimer)
  const fromQuery = Number(route.query.site)
  siteId.value = sites.value.some((site) => site.id === fromQuery) ? fromQuery : sites.value[0]?.id
  const fromStatus = String(route.query.status || '')
  status.value = statusOptions.some((option) => option.value === fromStatus)
    ? (fromStatus as Status)
    : 'pending'
  const fromType = String(route.query.type || '')
  typeFilter.value = typeOptions.some((option) => option.value === fromType)
    ? (fromType as TypeFilter)
    : 'all'
  const fromKeyword = String(route.query.q || '')
  searchInput.value = fromKeyword
  keyword.value = fromKeyword.trim()
  lastQueryKey = queryKey()
}

function syncQuery(): void {
  if (!siteId.value) return
  const query: Record<string, string> = { site: String(siteId.value), status: status.value }
  if (typeFilter.value !== 'all') query.type = typeFilter.value
  if (keyword.value) query.q = keyword.value
  void router.replace({ path: '/moderation', query })
}

async function load(reset = true): Promise<void> {
  if (!siteId.value) return
  if (reset) {
    annotations.value = []
    comments.value = []
    annotationCursor.value = 0
    commentCursor.value = 0
    selectedAnnotations.value = []
    selectedComments.value = []
    loading.value = true
  } else {
    if (!hasMore.value) return
    loadingMore.value = true
  }
  error.value = ''
  try {
    const wantAnnotations = reset ? typeFilter.value !== 'comment' : annotationCursor.value > 0
    const wantComments = reset ? typeFilter.value !== 'annotation' : commentCursor.value > 0
    const requestType = reset
      ? typeFilter.value
      : wantAnnotations && wantComments
        ? 'all'
        : wantAnnotations
          ? 'annotation'
          : 'comment'
    const queue: Queue = await api.queue(
      siteId.value,
      status.value,
      keyword.value,
      requestType,
      reset ? 0 : annotationCursor.value,
      reset ? 0 : commentCursor.value,
      50,
    )
    if (reset) {
      annotations.value = queue.annotations
      comments.value = queue.comments
    } else {
      if (wantAnnotations) annotations.value = [...annotations.value, ...queue.annotations]
      if (wantComments) comments.value = [...comments.value, ...queue.comments]
    }
    if (wantAnnotations) annotationCursor.value = queue.annotationCursor
    if (wantComments) commentCursor.value = queue.commentCursor
    counts.value = queue.counts
    if (reset) scheduleModerationBadgeRefresh()
  } catch (err: any) {
    error.value = err?.message || '加载审核列表失败'
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function actionsFor(raw: RowLike): string[] {
  const item = raw as Annotation | Comment
  if (item.visibility === 'hidden') return ['restore']
  if (item.moderation === 'pending') return ['approve', 'reject']
  if (item.moderation === 'approved') return ['hide']
  if (item.moderation === 'rejected') return ['approve']
  return []
}

function commonActions(items: Array<Annotation | ModerationComment>): string[] {
  if (items.length === 0) return []
  return actionsFor(items[0]).filter((action) => items.every((item) => actionsFor(item).includes(action)))
}

function annotationBody(raw: RowLike): string {
  const row = raw as Annotation
  return row.comments?.[0]?.content || '（无正文）'
}

function replyCount(raw: RowLike): number {
  const row = raw as Annotation
  return Math.max(0, (row.comments?.length || 0) - 1)
}

function pageLink(raw: RowLike): string {
  const row = raw as AnnotationBrief
  return buildPageLink(row.pageKey, currentSite.value?.origins || [])
}

function commentPageLink(raw: RowLike): string {
  const row = raw as ModerationComment
  return row.parent ? pageLink(row.parent) : ''
}

function commentPageTitle(raw: RowLike): string {
  const row = raw as ModerationComment
  return row.parent?.pageTitle || row.parent?.pageKey || '—'
}

function commentPageKey(raw: RowLike): string {
  const row = raw as ModerationComment
  return row.parent?.pageKey || ''
}

function parentBody(raw: RowLike): string {
  return (raw as ModerationComment).parent?.body || ''
}

function parentWarning(raw: RowLike): string {
  const parent = (raw as ModerationComment).parent
  if (!parent) return '所属批注已不存在'
  if (parent.visibility === 'deleted') return '所属批注已删除'
  if (parent.visibility === 'hidden') return '所属批注已隐藏'
  if (parent.moderation === 'pending') return '所属批注仍在待审'
  if (parent.moderation === 'rejected') return '所属批注未通过'
  return ''
}

function emptyText(kind: 'annotation' | 'comment'): string {
  const label = kind === 'annotation' ? '批注' : '评论'
  if (keyword.value) return `没有匹配的${label}`
  if (status.value === 'pending') return `暂无待审${label}`
  if (status.value === 'approved') return `暂无已通过${label}`
  return `暂无已拒绝${label}`
}

function shiftCount(kind: 'annotation' | 'comment', from: string, to: string): void {
  if (from === to) return
  const bucket = kind === 'annotation' ? counts.value.annotations : counts.value.comments
  if (from === 'pending' || from === 'approved' || from === 'rejected') {
    bucket[from] = Math.max(0, bucket[from] - 1)
  }
  if (to === 'pending' || to === 'approved' || to === 'rejected') {
    bucket[to] += 1
  }
}

function applyLocal(kind: 'annotation' | 'comment', id: number, action: string): void {
  if (kind === 'annotation') {
    if (action === 'hide' || action === 'restore') {
      annotations.value = annotations.value.map((item) =>
        item.id === id ? { ...item, visibility: action === 'hide' ? 'hidden' : 'visible' } : item,
      )
    } else {
      annotations.value = annotations.value.filter((item) => item.id !== id)
    }
    return
  }
  comments.value = comments.value.filter((item) => item.id !== id)
}

function onAnnotationSelection(rows: Annotation[]): void {
  selectedAnnotations.value = rows
}

function onCommentSelection(rows: ModerationComment[]): void {
  selectedComments.value = rows
}

async function openContext(raw: RowLike): Promise<void> {
  if (!siteId.value) return
  const row = raw as Annotation | ModerationComment
  const comment = row as ModerationComment
  const annotationId = comment.annotationId || (row as Annotation).id
  contextFocusId.value = comment.annotationId ? comment.id : null
  contextTitle.value = comment.annotationId ? '评论上下文' : '批注上下文'
  contextVisible.value = true
  contextLoading.value = true
  contextError.value = ''
  contextAnnotation.value = null
  try {
    contextAnnotation.value = await api.annotationDetail(siteId.value, annotationId)
    await nextTick()
    document.querySelector('.el-drawer .thread-item.focus')?.scrollIntoView({ block: 'center' })
  } catch (err: any) {
    contextError.value = err?.message || '加载上下文失败'
  } finally {
    contextLoading.value = false
  }
}

async function banAuthor(raw: RowLike): Promise<void> {
  const row = raw as ModerationComment
  if (!siteId.value || !row.author?.id) return
  if (auth.me?.id === row.author.id) {
    ElMessage.warning('不能封禁自己')
    return
  }
  try {
    const { value } = await ElMessageBox.prompt(
      `封禁「${row.author.name}」后，该用户将无法在本站点发布批注和评论。`,
      '封禁作者',
      {
        confirmButtonText: '封禁',
        cancelButtonText: '取消',
        inputPlaceholder: '封禁原因（可选）',
        inputValue: '',
        type: 'warning',
      },
    )
    await api.banSiteUser(siteId.value, row.author.id, value || '')
    ElMessage.success('已封禁')
  } catch (err: any) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error(err?.message || '封禁失败')
  }
}

async function moderate(
  kind: 'annotation' | 'comment',
  id: number,
  action: string,
  skipConfirm = false,
): Promise<void> {
  if (!skipConfirm) {
    try {
      await ElMessageBox.confirm(`确定对 #${id} 执行「${actionLabels[action]}」？`, '确认', {
        type: action === 'hide' || action === 'reject' ? 'warning' : 'info',
      })
    } catch {
      return
    }
  }
  const item =
    kind === 'annotation'
      ? annotations.value.find((row) => row.id === id)
      : comments.value.find((row) => row.id === id)
  try {
    if (kind === 'annotation') {
      await api.moderateAnnotation(siteId.value!, id, action)
    } else {
      await api.moderateComment(siteId.value!, id, action)
    }
    if (item) {
      const target =
        kind === 'annotation'
          ? action === 'approve'
            ? 'approved'
            : action === 'reject'
              ? 'rejected'
              : ''
          : action === 'approve' || action === 'restore'
            ? 'approved'
            : action === 'reject' || action === 'hide'
              ? 'rejected'
              : ''
      if (target) shiftCount(kind, item.moderation, target)
    }
    applyLocal(kind, id, action)
    scheduleModerationBadgeRefresh()
    ElMessage.success('已处理')
  } catch (err: any) {
    ElMessage.error(err?.message || '操作失败')
  }
}

async function bulk(kind: 'annotation' | 'comment', action: string): Promise<void> {
  const items = kind === 'annotation' ? selectedAnnotations.value : selectedComments.value
  if (items.length === 0) return
  const label = kind === 'annotation' ? '批注' : '评论'
  try {
    await ElMessageBox.confirm(`确定对 ${items.length} 条${label}执行「${actionLabels[action]}」？`, '批量操作', {
      type: action === 'reject' || action === 'hide' ? 'warning' : 'info',
    })
  } catch {
    return
  }
  bulkLoading.value = true
  try {
    const results = await Promise.allSettled(
      items.map((item) =>
        kind === 'annotation'
          ? api.moderateAnnotation(siteId.value!, item.id, action)
          : api.moderateComment(siteId.value!, item.id, action),
      ),
    )
    const failed = results.filter((result) => result.status === 'rejected').length
    if (failed > 0) ElMessage.warning(`完成，${failed} 条失败`)
    else ElMessage.success(`已处理 ${items.length} 条`)
    await load(true)
  } finally {
    bulkLoading.value = false
  }
}

watch([siteId, status, typeFilter, keyword], () => {
  const key = queryKey()
  if (key === lastQueryKey) return
  lastQueryKey = key
  void load(true)
  syncQuery()
})

watch(
  () => route.query,
  () => {
    const before = queryKey()
    initFromRoute()
    if (queryKey() !== before) void load(true)
  },
)

watch(searchInput, (value) => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    keyword.value = value.trim()
  }, 300)
})

onMounted(() => {
  initFromRoute()
  if (siteId.value) void load(true)
})

onBeforeUnmount(() => {
  window.clearTimeout(searchTimer)
})
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>内容审核</h1>
        <p>评论与批注的审核工作台，支持上下文预览、搜索与批量处理</p>
      </div>
      <el-button :loading="loading" @click="load(true)">刷新</el-button>
    </div>

    <el-empty v-if="sites.length === 0" description="暂无可管理站点，请联系超级管理员分配" />
    <template v-else>
      <div class="toolbar">
        <SiteSelector v-model="siteId" :sites="sites" />
        <el-radio-group v-model="status">
          <el-radio-button v-for="option in statusOptions" :key="option.value" :value="option.value">
            {{ option.label }}
            <span class="tab-count">{{ statusTotals[option.value] }}</span>
          </el-radio-button>
        </el-radio-group>
        <el-radio-group v-model="typeFilter">
          <el-radio-button v-for="option in typeOptions" :key="option.value" :value="option.value">
            {{ option.label }}
            <span class="tab-count">{{ typeTotals[option.value] }}</span>
          </el-radio-button>
        </el-radio-group>
        <el-input
          v-model="searchInput"
          placeholder="搜索评论、作者、引用、页面"
          clearable
          style="width: 260px"
        />
      </div>

      <div v-if="status === 'pending'" class="queue-hint">
        <span>
          当前站点待审 <strong>{{ statusTotals.pending }}</strong> 条
          <span class="muted">批注 {{ counts.annotations.pending }} · 评论 {{ counts.comments.pending }}</span>
        </span>
        <span class="muted">按提交时间从早到晚处理</span>
      </div>

      <ErrorState v-if="error" :message="error" @retry="load(true)" />
      <template v-else>
        <div v-loading="loading">
          <template v-if="typeFilter !== 'comment'">
            <div class="section-title">
              <span>批注 <span class="count">{{ annotations.length }}</span></span>
              <div v-if="selectedAnnotations.length" class="bulk">
                <span class="muted">已选 {{ selectedAnnotations.length }} 条</span>
                <span v-if="annotationBulkActions.length === 0" class="muted">所选状态不一致，请分开处理</span>
                <el-button
                  v-for="action in annotationBulkActions"
                  :key="action"
                  size="small"
                  :type="action === 'reject' || action === 'hide' ? 'danger' : 'success'"
                  :loading="bulkLoading"
                  @click="bulk('annotation', action)"
                >
                  批量{{ actionLabels[action] }}
                </el-button>
              </div>
            </div>
            <el-table
              :data="annotations"
              border
              row-key="id"
              @selection-change="onAnnotationSelection"
            >
              <el-table-column type="selection" width="44" />
              <el-table-column label="内容" min-width="380">
                <template #default="{ row }">
                  <div class="content-cell">
                    <blockquote class="quote">「{{ row.quoteExact }}」</blockquote>
                    <p class="body">{{ annotationBody(row) }}</p>
                    <div class="sub">
                      <span>{{ row.author.name }}</span>
                      <span>·</span>
                      <span>{{ formatTime(row.createdAt) }}</span>
                      <span v-if="replyCount(row)">· {{ replyCount(row) }} 条回复</span>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="页面" width="220">
                <template #default="{ row }">
                  <div class="page-cell">
                    <el-link
                      v-if="pageLink(row)"
                      :href="pageLink(row)"
                      target="_blank"
                      type="primary"
                    >
                      {{ row.pageTitle || row.pageKey }}
                    </el-link>
                    <span v-else>{{ row.pageTitle || row.pageKey }}</span>
                    <span class="page-key">{{ row.pageKey }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="96">
                <template #default="{ row }">
                  <StatusTag :status="row.visibility === 'hidden' ? 'hidden' : row.moderation" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="210" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openContext(row)">详情</el-button>
                  <el-button
                    v-for="action in actionsFor(row)"
                    :key="action"
                    link
                    :type="action === 'reject' || action === 'hide' ? 'danger' : 'primary'"
                    @click="moderate('annotation', row.id, action, action === 'approve' || action === 'restore')"
                  >
                    {{ actionLabels[action] }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <EmptyState
              v-if="!loading && annotations.length === 0"
              :description="emptyText('annotation')"
            />
          </template>

          <template v-if="typeFilter !== 'annotation'">
            <div class="section-title" :class="{ spaced: typeFilter === 'all' }">
              <span>评论 <span class="count">{{ comments.length }}</span></span>
              <div v-if="selectedComments.length" class="bulk">
                <span class="muted">已选 {{ selectedComments.length }} 条</span>
                <span v-if="commentBulkActions.length === 0" class="muted">所选状态不一致，请分开处理</span>
                <el-button
                  v-for="action in commentBulkActions"
                  :key="action"
                  size="small"
                  :type="action === 'reject' || action === 'hide' ? 'danger' : 'success'"
                  :loading="bulkLoading"
                  @click="bulk('comment', action)"
                >
                  批量{{ actionLabels[action] }}
                </el-button>
              </div>
            </div>
            <el-table
              :data="comments"
              border
              row-key="id"
              @selection-change="onCommentSelection"
            >
              <el-table-column type="selection" width="44" />
              <el-table-column label="评论" min-width="380">
                <template #default="{ row }">
                  <div class="content-cell">
                    <p class="body">{{ row.content }}</p>
                    <blockquote v-if="row.parent" class="quote">「{{ row.parent.quoteExact }}」</blockquote>
                    <p v-if="parentBody(row)" class="parent-body">
                      批注正文：{{ parentBody(row) }}
                    </p>
                    <div class="sub">
                      <span>{{ row.author.name }}</span>
                      <span>·</span>
                      <span>{{ formatTime(row.createdAt) }}</span>
                      <span>·</span>
                      <span>批注 #{{ row.annotationId }}</span>
                      <el-tag v-if="parentWarning(row)" size="small" type="warning" effect="plain">
                        {{ parentWarning(row) }}
                      </el-tag>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="所属页面" width="220">
                <template #default="{ row }">
                  <div class="page-cell">
                    <el-link
                      v-if="commentPageLink(row)"
                      :href="commentPageLink(row)"
                      target="_blank"
                      type="primary"
                    >
                      {{ commentPageTitle(row) }}
                    </el-link>
                    <span v-else>{{ commentPageTitle(row) }}</span>
                    <span class="page-key">{{ commentPageKey(row) }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="96">
                <template #default="{ row }">
                  <StatusTag :status="row.visibility === 'hidden' ? 'hidden' : row.moderation" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="260" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openContext(row)">上下文</el-button>
                  <el-button link type="danger" @click="banAuthor(row)">封禁</el-button>
                  <el-button
                    v-for="action in actionsFor(row)"
                    :key="action"
                    link
                    :type="action === 'reject' || action === 'hide' ? 'danger' : 'primary'"
                    @click="moderate('comment', row.id, action, action === 'approve' || action === 'restore')"
                  >
                    {{ actionLabels[action] }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <EmptyState
              v-if="!loading && comments.length === 0"
              :description="emptyText('comment')"
            />
          </template>
        </div>

        <LoadMore v-if="hasMore" :has-more="hasMore" :loading="loadingMore" @load="load(false)" />
      </template>
    </template>

    <el-drawer v-model="contextVisible" :title="contextTitle" size="520px">
      <div v-loading="contextLoading" class="context-body">
        <el-alert v-if="contextError" :title="contextError" type="error" :closable="false" />
        <template v-else-if="contextAnnotation">
          <div class="context-head">
            <StatusTag
              :status="contextAnnotation.visibility === 'hidden' ? 'hidden' : contextAnnotation.moderation"
            />
            <el-link
              v-if="pageLink(contextAnnotation)"
              :href="pageLink(contextAnnotation)"
              target="_blank"
              type="primary"
            >
              查看原文
            </el-link>
          </div>
          <blockquote class="quote">「{{ contextAnnotation.quoteExact }}」</blockquote>
          <div class="context-meta">
            <span>{{ contextAnnotation.pageTitle || contextAnnotation.pageKey }}</span>
            <span>· {{ contextAnnotation.author.name }}</span>
            <span>· {{ formatTime(contextAnnotation.createdAt) }}</span>
          </div>
          <div class="thread">
            <div
              v-for="(comment, index) in contextAnnotation.comments || []"
              :key="comment.id"
              class="thread-item"
              :class="{ focus: comment.id === contextFocusId }"
            >
              <div class="thread-meta">
                <span class="thread-author">{{ comment.author.name }}</span>
                <span>{{ formatTime(comment.createdAt) }}</span>
                <el-tag v-if="index === 0" size="small" effect="plain">批注正文</el-tag>
                <StatusTag v-if="comment.moderation !== 'approved'" :status="comment.moderation" />
              </div>
              <p class="body">{{ comment.content }}</p>
            </div>
            <EmptyState v-if="(contextAnnotation.comments || []).length === 0" description="暂无讨论内容" />
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.tab-count {
  margin-left: 4px;
  padding: 0 5px;
  border-radius: 8px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.queue-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  padding: 10px 14px;
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: 8px;
  background: var(--el-color-warning-light-9);
  font-size: 13px;
}
.queue-hint strong {
  margin: 0 2px;
  color: var(--el-color-warning-dark-2);
  font-size: 16px;
}
.queue-hint .muted {
  margin-left: 8px;
}
.count {
  margin-left: 4px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  font-weight: 400;
}
.bulk {
  display: flex;
  align-items: center;
  gap: 8px;
}
.section-title.spaced {
  margin-top: 24px;
}
.content-cell {
  padding: 2px 0;
}
.quote {
  margin: 6px 0;
  padding: 4px 8px;
  border-left: 3px solid var(--el-color-primary);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.6;
}
.body {
  margin: 0;
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}
.parent-body {
  display: -webkit-box;
  margin: 2px 0 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.page-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 13px;
}
.page-key {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.context-body {
  min-height: 160px;
}
.context-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.context-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 16px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.thread {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.thread-item {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}
.thread-item.focus {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
}
.thread-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.thread-author {
  color: var(--el-text-color-primary);
  font-weight: 500;
}
</style>
