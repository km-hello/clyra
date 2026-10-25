<script setup lang="ts">
import { ref, computed, nextTick, useTemplateRef, watch } from 'vue'
import { NButton, NPopover } from 'naive-ui'
import { Bell, ChevronDown, ChevronRight, LockKeyhole, LogOut, Sun } from '@lucide/vue'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useApi } from '@/composables/useApi'
import { useUserSession } from '@/composables/useUserSession'
import { useThemeMode } from '@/composables/useThemeMode'
import { useRouter } from 'vue-router'
const router = useRouter()
import type { SettingsValues, SettingsResponse } from '@/types/settings'

const emit = defineEmits<{ profile: []; settings: []; password: []; saved: [] }>()
const $api = useApi()
const session = useUserSession()
const colorMode = useThemeMode()
const open = ref(false)
const trigger = useTemplateRef('trigger')
const dialog = useTemplateRef<HTMLDivElement>('dialog')
watch(open, async (value) => {
  await nextTick()
  if (value) dialog.value?.focus()
  else trigger.value?.$el.focus()
})
function trapTab(event: KeyboardEvent) {
  const buttons = [
    ...(dialog.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') || []),
  ]
  const first = buttons[0]
  const last = buttons.at(-1)
  if (
    event.shiftKey &&
    (document.activeElement === first || document.activeElement === dialog.value)
  ) {
    event.preventDefault()
    last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus()
  }
}
const savingTheme = ref(false)
const loggingOut = ref(false)
const errorMessage = ref('')
const busy = computed(() => savingTheme.value || loggingOut.value)
const displayName = computed(
  () => session.user.value?.nickname || session.user.value?.username || '',
)
const themes = [
  { label: '跟随系统', value: 'system' },
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
] as const

async function showProfile() {
  open.value = false
  await nextTick()
  trigger.value?.$el.focus()
  emit('profile')
}

async function showSettings() {
  open.value = false
  await nextTick()
  trigger.value?.$el.focus()
  emit('settings')
}
async function showPassword() {
  open.value = false
  await nextTick()
  trigger.value?.$el.focus()
  emit('password')
}
async function setTheme(theme: SettingsValues['theme']) {
  if (busy.value || theme === colorMode.preference) return
  savingTheme.value = true
  errorMessage.value = ''
  try {
    const response = await $api<{ data: SettingsResponse }>('/api/settings', {
      method: 'PATCH',
      body: { theme },
      retry: false,
    })
    colorMode.preference = response.data.theme
    emit('saved')
  } catch {
    errorMessage.value = '主题保存失败，请重试'
  } finally {
    savingTheme.value = false
  }
}
async function logout(event: MouseEvent) {
  if (busy.value) return
  // Keep focus in the menu before its last enabled control becomes disabled.
  ;(event.currentTarget as HTMLElement).closest<HTMLElement>('[role="dialog"]')?.focus()
  loggingOut.value = true
  errorMessage.value = ''
  try {
    await session.clear()
    colorMode.preference = 'system'
    await router.push('/login')
  } catch {
    errorMessage.value = '退出失败，请重试'
  } finally {
    loggingOut.value = false
  }
}
</script>

<template>
  <NPopover
    :show="open"
    trigger="click"
    placement="bottom-end"
    :show-arrow="false"
    :style="{ padding: '8px', width: '320px', maxHeight: 'calc(100dvh - 96px)', overflowY: 'auto' }"
    @update:show="
      (value) => {
        if (!loggingOut) open = value
      }
    "
  >
    <template #trigger>
      <NButton
        ref="trigger"
        quaternary
        :theme-overrides="{
          colorQuaternaryHover: 'color-mix(in srgb, var(--ui-primary) 10%, transparent)',
          colorQuaternaryPressed: 'color-mix(in srgb, var(--ui-primary) 16%, transparent)',
        }"
        :disabled="loggingOut"
        :aria-label="`账户菜单：${displayName}`"
        :aria-expanded="open"
        aria-haspopup="dialog"
        class="px-2 sm:px-3"
      >
        <UserAvatar :src="session.user.value?.avatar_url" />
        <span class="max-w-24 truncate sm:max-w-32">{{ displayName }}</span>
        <ChevronDown class="size-4" />
      </NButton>
    </template>
    <div
      ref="dialog"
      class="outline-none"
      role="dialog"
      aria-modal="true"
      aria-label="账户菜单"
      tabindex="-1"
      @keydown.tab="trapTab"
      @keydown.esc="
        () => {
          if (!loggingOut) {
            open = false
            trigger?.$el.focus()
          }
        }
      "
    >
      <button
        type="button"
        class="account-menu-item account-identity"
        :disabled="busy"
        aria-label="编辑资料"
        @click="showProfile"
      >
        <UserAvatar :src="session.user.value?.avatar_url" :size="48" />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-base font-semibold">{{ displayName }}</span>
          <span
            v-if="displayName !== session.user.value?.username"
            class="mt-1 block truncate text-sm font-normal text-muted"
            >@{{ session.user.value?.username }}</span
          >
        </span>
        <span class="text-xs font-medium text-muted">编辑资料</span>
        <ChevronRight class="size-4 shrink-0 text-muted" />
      </button>
      <div class="mt-2 border-t border-default pt-2">
        <button type="button" class="account-menu-item" :disabled="busy" @click="showSettings">
          <Bell class="size-5 shrink-0" /><span>提醒设置</span
          ><ChevronRight class="ms-auto size-4 text-muted" />
        </button>
        <div class="px-3 pt-3 pb-4" :aria-busy="savingTheme">
          <p class="mb-3 flex items-center gap-3 text-sm font-semibold">
            <Sun class="size-5 shrink-0" />主题模式
          </p>
          <div class="flex gap-2" role="group" aria-label="主题模式">
            <NButton
              v-for="theme in themes"
              :key="theme.value"
              :aria-pressed="colorMode.preference === theme.value"
              :disabled="busy"
              class="theme-choice flex-1 px-2 font-medium"
              @click="setTheme(theme.value)"
              >{{ theme.label }}</NButton
            >
          </div>
        </div>
      </div>
      <p v-if="errorMessage" class="px-3 pb-3 text-sm text-error" role="alert">
        {{ errorMessage }}
      </p>
      <div class="border-t border-default pt-2" role="group" aria-label="账户操作">
        <button type="button" class="account-menu-item" :disabled="busy" @click="showPassword">
          <LockKeyhole class="size-5 shrink-0" /><span>修改密码</span
          ><ChevronRight class="ms-auto size-4 text-muted" />
        </button>
        <button
          type="button"
          class="account-menu-item danger"
          :disabled="busy"
          :aria-busy="loggingOut"
          @click="logout"
        >
          <LogOut class="size-5 shrink-0" />{{ loggingOut ? '正在退出…' : '退出登录' }}
        </button>
      </div>
    </div>
  </NPopover>
</template>

<style scoped>
.theme-choice[aria-pressed='true'] {
  background: color-mix(in srgb, var(--ui-primary) 8%, var(--ui-bg-elevated));
  color: var(--ui-primary);
}

.theme-choice[aria-pressed='true'] :deep(.n-button__border),
.theme-choice[aria-pressed='true'] :deep(.n-button__state-border) {
  border-color: var(--ui-primary);
  box-shadow: none;
}
</style>
