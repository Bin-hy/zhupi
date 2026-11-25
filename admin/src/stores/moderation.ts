import { reactive } from 'vue'
import { api } from '../api'

/** 侧边栏待审角标：跨站点汇总，审核动作后轻量刷新。 */
export const moderationBadge = reactive({
  pending: 0,
})

let seq = 0
let timer: number | undefined

export async function refreshModerationBadge(): Promise<void> {
  const current = ++seq
  try {
    const overview = await api.overview()
    if (current !== seq) return
    moderationBadge.pending =
      (overview.counts.pendingAnnotations || 0) + (overview.counts.pendingComments || 0)
  } catch {
    /* 角标失败不影响主流程 */
  }
}

export function scheduleModerationBadgeRefresh(delay = 300): void {
  window.clearTimeout(timer)
  timer = window.setTimeout(() => void refreshModerationBadge(), delay)
}
