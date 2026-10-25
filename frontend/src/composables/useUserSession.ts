import { computed, ref } from 'vue'
import { ofetch } from 'ofetch'
import type { User } from '@/types/auth'

const session = ref<{ id?: string; user?: User }>({})
const ready = ref(false)
const user = computed(() => session.value.user)
const loggedIn = computed(() => !!user.value)

export function useUserSession() {
  return {
    session,
    user,
    ready,
    loggedIn,
    async fetch() {
      session.value = await ofetch('/api/_auth/session', { retry: false })
      ready.value = true
    },
    async clear() {
      await ofetch('/api/_auth/session', { method: 'DELETE', retry: false })
      session.value = {}
    },
  }
}
