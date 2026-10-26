<script setup lang="ts">
import { ref, computed, watch, h } from 'vue'
import { NButton, NCard, NDropdown, NProgress, NTooltip, type DropdownOption } from 'naive-ui'
import {
  BellOff,
  CirclePause,
  Ellipsis,
  ExternalLink,
  Pause,
  Pencil,
  Play,
  RefreshCw,
  Trash2,
} from '@lucide/vue'
import type { Subscription } from '@/types/subscriptions'
import { currentCycle, dayNumber } from '@/utils/dates'
import { formatAmountMinor } from '@/utils/money'
import { isSubscriptionLink } from '@/utils/subscriptionLink'

const props = defineProps<{ subscription: Subscription; today: string; reminderDays: number }>()
const emit = defineEmits<{
  edit: []
  deactivate: []
  activate: []
  delete: []
  renew: []
}>()
const imageFailed = ref(false)
const statusTooltipOpen = ref(false)
const progressTooltipOpen = ref(false)
const statusLabel = computed(() =>
  props.subscription.status === 'inactive'
    ? '已停用'
    : !props.subscription.notify_enabled
      ? '提醒已关闭'
      : null,
)
watch(statusLabel, () => (statusTooltipOpen.value = false))
const websiteUrl = computed(() => {
  const value = props.subscription.website_url
  return value && isSubscriptionLink(value) ? value : null
})
const imageIcon = computed(() => /^https:\/\//i.test(props.subscription.icon ?? ''))
watch(
  () => props.subscription.icon,
  () => (imageFailed.value = false),
)
const days = computed(() => dayNumber(props.subscription.next_payment) - dayNumber(props.today))
const units = { day: '天', week: '周', month: '个月', year: '年' }
const cycle = computed(() =>
  props.subscription.subscription_type === 'one_time'
    ? '一次性'
    : `每 ${props.subscription.interval_count} ${units[props.subscription.interval_unit!]}`,
)
const dueText = computed(() => {
  if (days.value === 0) return '今天'
  return `${Math.abs(days.value)} 天`
})
const dueLabel = computed(() => (days.value < 0 ? '已过期' : days.value === 0 ? '到期' : '后到期'))
const progressColor = computed(() => {
  if (props.subscription.status === 'inactive') return 'var(--ui-border)'
  if (days.value <= 0) return 'var(--ui-error)'
  if (days.value <= (props.subscription.reminder_days ?? props.reminderDays))
    return 'var(--ui-progress-warning)'
  return 'var(--ui-progress)'
})
const progress = computed(() => {
  const item = props.subscription
  if (item.subscription_type !== 'recurring' || !item.start_date) return null
  return currentCycle(
    item.start_date,
    item.next_payment,
    item.interval_count!,
    item.interval_unit!,
    props.today,
  )
})
const menu = computed<DropdownOption[]>(() => [
  { label: '编辑订阅', key: 'edit', icon: () => h(Pencil, { size: 18 }) },
  ...(props.subscription.status === 'active' && props.subscription.subscription_type === 'recurring'
    ? [{ label: '续订一次', key: 'renew', icon: () => h(RefreshCw, { size: 18 }) }]
    : []),
  props.subscription.status === 'active'
    ? { label: '停用订阅', key: 'deactivate', icon: () => h(Pause, { size: 18 }) }
    : { label: '恢复订阅', key: 'activate', icon: () => h(Play, { size: 18 }) },
  { type: 'divider', key: 'divider' },
  {
    label: () => h('span', { style: { color: 'var(--ui-error)' } }, '删除订阅'),
    key: 'delete',
    icon: () => h(Trash2, { size: 18, color: 'var(--ui-error)' }),
  },
])
function selectAction(key: string) {
  if (key === 'edit') emit('edit')
  else if (key === 'renew') emit('renew')
  else if (key === 'deactivate') emit('deactivate')
  else if (key === 'activate') emit('activate')
  else if (key === 'delete') emit('delete')
}
</script>

<template>
  <NCard
    class="subscription-card relative p-5 sm:px-6"
    :content-style="{ display: 'flex', flex: '1', padding: '0' }"
    :aria-label="subscription.name"
  >
    <button
      type="button"
      class="absolute inset-0 z-1 cursor-pointer rounded-2xl outline-none ring-inset hover:ring-1 hover:ring-primary/40 focus-visible:ring-2 focus-visible:ring-primary"
      :aria-label="`编辑${subscription.name}`"
      @click="emit('edit')"
    />
    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex items-start gap-4">
        <div class="subscription-icon" aria-hidden="true">
          <img
            v-if="imageIcon && !imageFailed"
            :src="subscription.icon ?? undefined"
            alt=""
            referrerpolicy="no-referrer"
            class="h-full w-full rounded-xl object-contain"
            @error="imageFailed = true"
          />
          <span v-else class="truncate px-1">{{
            !imageIcon && subscription.icon ? subscription.icon : Array.from(subscription.name)[0]
          }}</span>
        </div>
        <div class="min-w-0 flex-1">
          <NTooltip
            :style="{ maxWidth: 'min(320px, calc(100vw - 32px))', overflowWrap: 'anywhere' }"
          >
            <template #trigger>
              <h2
                class="relative z-2 cursor-pointer truncate text-lg leading-6 font-semibold"
                @click="emit('edit')"
              >
                <a
                  v-if="websiteUrl"
                  :href="websiteUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  :aria-label="`${subscription.name}（在新标签页打开）`"
                  class="group flex min-w-0 items-center gap-1 rounded-sm hover:text-primary hover:underline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-primary"
                  @click.stop
                  ><span class="min-w-0 truncate">{{ subscription.name }}</span>
                  <ExternalLink
                    :size="14"
                    aria-hidden="true"
                    class="shrink-0 opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100"
                  />
                </a>
                <template v-else>{{ subscription.name }}</template>
              </h2>
            </template>
            {{ subscription.name }}
          </NTooltip>
          <p class="mt-1 flex flex-wrap items-baseline gap-x-1 text-sm leading-5 text-muted">
            <span class="max-w-full [overflow-wrap:anywhere]">{{ cycle }}</span>
            <span class="min-w-0 max-w-full break-all tabular-nums"
              >/ {{ formatAmountMinor(subscription.amount_minor, true) }}
              {{ subscription.currency }}</span
            >
          </p>
        </div>
        <div class="relative z-2 -me-2 -mt-2 flex shrink-0 items-center">
          <NTooltip v-if="statusLabel" v-model:show="statusTooltipOpen">
            <template #trigger>
              <button
                type="button"
                :aria-label="statusLabel"
                class="flex h-11 w-7 cursor-help items-center justify-center rounded-sm text-muted focus-visible:outline-2 focus-visible:outline-primary"
                @focus="statusTooltipOpen = true"
                @blur="statusTooltipOpen = false"
                @click.stop="statusTooltipOpen = true"
                @keydown.esc.stop="statusTooltipOpen = false"
              >
                <CirclePause
                  v-if="subscription.status === 'inactive'"
                  :size="16"
                  aria-hidden="true"
                />
                <BellOff v-else :size="16" aria-hidden="true" />
              </button>
            </template>
            {{ statusLabel }}
          </NTooltip>
          <NDropdown trigger="click" :options="menu" @select="selectAction">
            <NButton quaternary :aria-label="`${subscription.name}的更多操作`"
              ><template #icon><Ellipsis :size="20" /></template
            ></NButton>
          </NDropdown>
        </div>
      </div>
      <div class="mt-auto grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 pt-4">
        <div class="min-w-0">
          <p class="text-sm text-muted">到期</p>
          <p class="mt-1 text-sm tabular-nums text-muted">{{ subscription.next_payment }}</p>
        </div>
        <div
          class="flex min-h-14 min-w-24 flex-col items-center justify-center border-s border-default ps-4 text-center"
        >
          <p
            class="text-[28px] leading-8 font-semibold tabular-nums"
            :class="
              subscription.status === 'inactive'
                ? 'text-muted'
                : days <= 0
                  ? 'text-error'
                  : days <= (subscription.reminder_days ?? reminderDays)
                    ? 'text-warning'
                    : 'text-highlighted'
            "
          >
            {{ dueText }}
          </p>
          <p class="mt-1 text-sm text-muted">{{ dueLabel }}</p>
        </div>
      </div>
      <div class="pt-3">
        <div class="h-1">
          <NTooltip v-if="progress" v-model:show="progressTooltipOpen">
            <template #trigger>
              <div
                class="relative z-2 rounded-sm focus-visible:outline-2 focus-visible:outline-primary"
                tabindex="0"
                role="progressbar"
                aria-label="当前周期进度"
                :aria-valuenow="progress.percentage"
                :aria-valuemin="0"
                :aria-valuemax="100"
                :aria-valuetext="`${progress.start} 至 ${progress.end}，已过 ${progress.percentage}%`"
                @focus="progressTooltipOpen = true"
                @blur="progressTooltipOpen = false"
                @keydown.esc.stop="progressTooltipOpen = false"
              >
                <NProgress
                  :percentage="progress.percentage"
                  :color="progressColor"
                  type="line"
                  :show-indicator="false"
                  :height="4"
                  aria-hidden="true"
                />
              </div>
            </template>
            {{ progress.start }} 至 {{ progress.end }} · {{ progress.percentage }}%
          </NTooltip>
        </div>
      </div>
    </div>
  </NCard>
</template>
