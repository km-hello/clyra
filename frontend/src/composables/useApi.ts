import { h } from 'vue'
import { ofetch } from 'ofetch'
import { NButton, useNotification, type NotificationReactive } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useUserSession } from '@/composables/useUserSession'

let recovery: Promise<void> | undefined
let expired: NotificationReactive | undefined
export function useApi() {
  const session = useUserSession()
  const notification = useNotification()
  const router = useRouter()
  return ofetch.create({
    async onResponseError({ response }) {
      if (response.status !== 401) return
      recovery ??= session.fetch().finally(() => {
        recovery = undefined
      })
      await recovery
      if (!session.loggedIn.value && !expired)
        expired = notification.error({
          title: '会话已失效，请重新登录',
          content: '当前输入已保留；重新登录会离开此页面。',
          duration: 0,
          onAfterLeave: () => {
            expired = undefined
          },
          action: () =>
            h(
              NButton,
              {
                type: 'primary',
                onClick: () => {
                  expired?.destroy()
                  void router.push('/login')
                },
              },
              { default: () => '重新登录' },
            ),
        })
    },
  })
}
