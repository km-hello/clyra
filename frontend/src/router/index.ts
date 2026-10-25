import { createRouter, createWebHistory } from 'vue-router'
import { ofetch } from 'ofetch'
import { useUserSession } from '@/composables/useUserSession'
import { useThemeMode } from '@/composables/useThemeMode'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/pages/HomePage.vue'),
      meta: { title: 'Clyra' },
    },
    {
      path: '/login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { title: '登录 · Clyra' },
    },
    {
      path: '/setup',
      component: () => import('@/pages/SetupPage.vue'),
      meta: { title: '首次初始化 · Clyra' },
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})
router.beforeEach(async (to) => {
  const session = useUserSession()
  if (!session.ready.value || to.path !== '/') await session.fetch()
  if (session.loggedIn.value) return to.path === '/' ? true : '/'
  useThemeMode().preference = 'system'
  const result = await ofetch<{ data: { needs_setup: boolean } }>('/api/setup/status', {
    retry: false,
  })
  const target = result.data.needs_setup ? '/setup' : '/login'
  return to.path === target ? true : target
})
router.afterEach((to) => {
  document.title = String(to.meta.title || 'Clyra')
})
