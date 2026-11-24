import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import AdminLayout from './layout/AdminLayout.vue'
import LoginView from './views/LoginView.vue'
import OverviewView from './views/OverviewView.vue'
import SitesView from './views/SitesView.vue'
import SiteDetailView from './views/SiteDetailView.vue'
import ModerationView from './views/ModerationView.vue'
import UsersView from './views/UsersView.vue'
import AccountView from './views/AccountView.vue'
import NotFoundView from './views/NotFoundView.vue'
import { auth, canAccessSite, hasToken, isSuperadmin, loadSession, resetSession } from './stores/auth'

const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
  {
    path: '/',
    component: AdminLayout,
    children: [
      { path: '', name: 'overview', component: OverviewView },
      { path: 'sites', name: 'sites', component: SitesView },
      { path: 'sites/:id', name: 'site-detail', component: SiteDetailView },
      { path: 'moderation', name: 'moderation', component: ModerationView },
      { path: 'users', name: 'users', component: UsersView, meta: { superadmin: true } },
      { path: 'account', name: 'account', component: AccountView },
    ],
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
]

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes,
})

async function ensureSession(): Promise<void> {
  if (!auth.loaded) await loadSession()
}

router.beforeEach(async (to) => {
  if (to.meta.public) {
    if (to.name === 'login' && hasToken()) {
      try {
        await ensureSession()
        return { path: auth.me?.mustChangePassword ? '/account' : '/' }
      } catch {
        resetSession()
      }
    }
    return true
  }
  if (!hasToken()) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  try {
    await ensureSession()
  } catch {
    resetSession()
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (auth.me?.mustChangePassword && to.name !== 'account') return { path: '/account' }
  if (to.meta.superadmin && !isSuperadmin()) return { path: '/' }
  if (to.name === 'site-detail' && !canAccessSite(Number(to.params.id))) return { path: '/' }
  return true
})

export default router
