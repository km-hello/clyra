<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, nextTick, useTemplateRef } from 'vue'
import { NAlert, NButton, NFormItem, NInput, NPopover, NSkeleton, useMessage } from 'naive-ui'
import { CalendarDays, Filter, Plus, Search } from '@lucide/vue'
import ClyraBrand from '@/components/common/ClyraBrand.vue'
import AccountMenu from '@/components/account/AccountMenu.vue'
import SubscriptionCard from '@/components/subscriptions/SubscriptionCard.vue'
import SubscriptionFormModal from '@/components/subscriptions/SubscriptionFormModal.vue'
import SubscriptionActionModal from '@/components/subscriptions/SubscriptionActionModal.vue'
import SettingsModal from '@/components/settings/SettingsModal.vue'
import PasswordModal from '@/components/account/PasswordModal.vue'
import ProfileModal from '@/components/account/ProfileModal.vue'
import { useApi } from '@/composables/useApi'
import { useThemeMode } from '@/composables/useThemeMode'
import { useRoute, useRouter } from 'vue-router'
import { useSubscriptionList } from '@/composables/useSubscriptionList'
import type { Subscription } from '@/types/subscriptions'

const $api = useApi()

const route = useRoute()
const router = useRouter()
const message = useMessage()
const colorMode = useThemeMode()
const filtersOpen = ref(false)
const filterTrigger = useTemplateRef('filterTrigger')
const filterPanel = useTemplateRef<HTMLDivElement>('filterPanel')
watch(filtersOpen, async (open) => {
  await nextTick()
  if (open) filterPanel.value?.querySelector<HTMLInputElement>('input:checked')?.focus()
  else filterTrigger.value?.$el.focus()
})
const settingsOpen = ref(false)
const passwordOpen = ref(false)
const profileOpen = ref(false)
const editorOpen = ref(false)
const editing = ref<Subscription | null>(null)
const actionOpen = ref(false)
const actionItem = ref<Subscription | null>(null)
const action = ref<'deactivate' | 'delete' | 'renew' | null>(null)
const restoring = ref<string | null>(null)
const search = ref(String(route.query.search || ''))
let searchTimer: ReturnType<typeof setTimeout> | undefined
const query = computed(() => ({
  ...route.query,
  status: route.query.status || 'all',
  page_size: 50,
}))
const { data, status, error, refresh } = useSubscriptionList(query)
const filtered = computed(
  () =>
    (!!route.query.status && route.query.status !== 'all') ||
    (!!route.query.due && route.query.due !== 'all') ||
    (!!route.query.sort && route.query.sort !== 'next_payment_asc'),
)
const stateOptions = [
  { label: '全部', value: 'all' },
  { label: '活跃', value: 'active' },
  { label: '已停用', value: 'inactive' },
]
const dueOptions = [
  { label: '全部', value: 'all' },
  { label: '已过期', value: 'overdue' },
  { label: '今天到期', value: 'today' },
  { label: '提醒窗口', value: 'window' },
  { label: '未来 30 天', value: 'next_30_days' },
]
const sortOptions = [
  { label: '到期近 → 远', value: 'next_payment_asc' },
  { label: '到期远 → 近', value: 'next_payment_desc' },
  { label: '名称 A → Z', value: 'name_asc' },
]
watch(
  () => route.query,
  (value) => {
    clearTimeout(searchTimer)
    search.value = String(value.search || '')
  },
)
watch(
  () => data.value?.meta,
  (meta) => {
    if (!meta) return
    colorMode.preference = meta.theme
    if (meta.page > meta.total_pages)
      void router.replace({ query: { ...route.query, page: String(meta.total_pages) } })
  },
  { immediate: true },
)
onBeforeUnmount(() => clearTimeout(searchTimer))

function setQuery(values: Record<string, string>) {
  clearTimeout(searchTimer)
  const next: typeof route.query = {
    ...route.query,
    search: search.value.trim(),
    page: '1',
    ...values,
  }
  if (String(next.search || '') !== String(route.query.search || '')) next.page = '1'
  const normalized = Object.fromEntries(
    Object.entries(next).filter(
      ([key, value]) =>
        !(
          value === '' ||
          (key === 'page' && value === '1') ||
          (key === 'status' && value === 'all') ||
          (key === 'due' && value === 'all') ||
          (key === 'sort' && value === 'next_payment_asc')
        ),
    ),
  )
  return router.push({ query: normalized })
}
async function resetFilters() {
  await setQuery({ status: 'all', due: 'all', sort: 'next_payment_asc' })
  await nextTick()
  filterPanel.value?.querySelector<HTMLInputElement>('input:checked')?.focus()
}
function searchChanged(value: string) {
  search.value = value
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => setQuery({ search: value.trim() }), 250)
}
function clearFilters() {
  clearTimeout(searchTimer)
  search.value = ''
  void router.push({ query: {} })
}
function edit(item: Subscription | null) {
  editing.value = item
  editorOpen.value = true
}
function confirmAction(item: Subscription, value: 'deactivate' | 'delete' | 'renew') {
  actionItem.value = item
  action.value = value
  actionOpen.value = true
}
function closeAction() {
  actionOpen.value = false
}
async function saved() {
  await refresh()
  await nextTick()
  document.getElementById('new-subscription')?.focus()
}
async function activate(item: Subscription) {
  if (restoring.value) return
  restoring.value = item.id
  try {
    await $api(`/api/subscriptions/${item.id}`, {
      method: 'PATCH',
      body: { expected_version: item.version, status: 'active' },
      retry: false,
    })
    await refresh()
    message.success(
      error.value
        ? '订阅已恢复；列表刷新失败，请重试'
        : data.value?.data.some((row) => row.id === item.id)
          ? '订阅已恢复'
          : '订阅已恢复，已移出当前筛选结果',
    )
  } catch (failure) {
    const response = failure as { statusCode?: number; data?: { message?: string } }
    message.error(
      response.statusCode === 409
        ? '订阅已更新，请刷新列表'
        : response.data?.message || '恢复失败，请重试',
    )
  } finally {
    restoring.value = null
  }
}
</script>

<template>
  <main class="home-container">
    <header class="home-header">
      <ClyraBrand />
      <div class="flex items-center gap-2">
        <AccountMenu
          @profile="profileOpen = true"
          @settings="settingsOpen = true"
          @password="passwordOpen = true"
          @saved="refresh()"
        />
      </div>
    </header>
    <div class="home-content">
      <div class="subscription-toolbar">
        <div class="subscription-heading flex min-w-0 items-center gap-3">
          <h1 class="text-2xl font-bold">我的订阅</h1>
          <span
            v-if="data"
            class="text-sm text-muted"
            :aria-label="`共 ${data.meta.total} 个订阅`"
            >{{ data.meta.total }}</span
          >
        </div>
        <div class="subscription-tools">
          <label class="sr-only" for="subscription-search">搜索订阅</label>
          <NInput
            :input-props="{ id: 'subscription-search', 'aria-label': '搜索订阅' }"
            :value="search"
            placeholder="搜索订阅"
            class="subscription-search"
            @update:value="searchChanged"
            ><template #prefix><Search class="size-4 text-muted" /></template
          ></NInput>
          <NPopover
            v-model:show="filtersOpen"
            trigger="click"
            placement="bottom-end"
            :show-arrow="false"
            :style="{ padding: '0' }"
          >
            <template #trigger
              ><NButton ref="filterTrigger" :aria-expanded="filtersOpen" aria-haspopup="dialog"
                ><template #icon><Filter class="size-5" /></template>筛选<span
                  v-if="filtered"
                  aria-label="已应用筛选"
                  class="size-1.5 rounded-full bg-primary" /></NButton
            ></template>
            <div
              ref="filterPanel"
              role="dialog"
              aria-label="筛选"
              class="w-[min(340px,calc(100vw-32px))] space-y-5 p-5"
              @keydown.esc.prevent="filtersOpen = false"
            >
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">筛选</h2>
                <NButton text type="primary" :disabled="!filtered" @click="resetFilters"
                  >重置</NButton
                >
              </div>
              <fieldset>
                <legend class="mb-2 text-sm font-semibold">状态</legend>
                <div class="filter-status flex">
                  <label v-for="option in stateOptions" :key="option.value" class="flex-1">
                    <input
                      type="radio"
                      name="filter-status"
                      class="sr-only"
                      :value="option.value"
                      :checked="(route.query.status || 'all') === option.value"
                      @change="setQuery({ status: option.value })"
                    />
                    <span class="filter-choice">{{ option.label }}</span>
                  </label>
                </div>
              </fieldset>
              <fieldset>
                <legend class="mb-2 text-sm font-semibold">到期范围</legend>
                <div class="flex flex-wrap gap-2">
                  <label v-for="option in dueOptions" :key="option.value">
                    <input
                      type="radio"
                      name="filter-due"
                      class="sr-only"
                      :value="option.value"
                      :checked="(route.query.due || 'all') === option.value"
                      @change="setQuery({ due: option.value })"
                    />
                    <span class="filter-choice rounded-full">{{ option.label }}</span>
                  </label>
                </div>
              </fieldset>
              <NFormItem label="排序" :label-props="{ for: 'filter-sort' }" :show-feedback="false"
                ><select
                  id="filter-sort"
                  class="native-control"
                  :value="String(route.query.sort || 'next_payment_asc')"
                  @change="setQuery({ sort: ($event.target as HTMLSelectElement).value })"
                >
                  <option v-for="option in sortOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select></NFormItem
              >
            </div>
          </NPopover>
        </div>
        <NButton
          id="new-subscription"
          class="subscription-create"
          type="primary"
          @click="edit(null)"
          ><template #icon><Plus class="size-5" /></template>新增订阅</NButton
        >
      </div>
      <NAlert
        v-if="error && data"
        type="error"
        title="刷新失败，当前显示上次结果"
        role="alert"
        class="mb-5"
      >
        <template #default><NButton @click="refresh()">重试</NButton></template>
      </NAlert>
      <div v-if="!data && status === 'pending'" class="subscription-grid" aria-label="正在加载订阅">
        <NSkeleton v-for="index in 6" :key="index" height="232px" :sharp="false" />
      </div>
      <div v-else-if="error && !data" class="subscription-empty">
        <h2 class="text-lg font-semibold">订阅加载失败</h2>
        <div class="flex justify-center gap-2">
          <NButton @click="refresh()">重试</NButton
          ><NButton @click="clearFilters">清除筛选</NButton>
        </div>
      </div>
      <div
        v-else-if="data?.data.length"
        class="subscription-grid"
        :aria-busy="status === 'pending' || !!restoring"
      >
        <SubscriptionCard
          v-for="item in data.data"
          :key="item.id"
          :subscription="item"
          :today="data.meta.today"
          :reminder-days="data.meta.default_reminder_days"
          @edit="edit(item)"
          @deactivate="confirmAction(item, 'deactivate')"
          @activate="activate(item)"
          @delete="confirmAction(item, 'delete')"
          @renew="confirmAction(item, 'renew')"
        />
      </div>
      <div v-else-if="data" class="subscription-empty">
        <CalendarDays class="mx-auto size-10 text-primary" />
        <h2 class="text-lg font-semibold">
          {{ data.meta.has_any_subscriptions ? '没有匹配的订阅' : '还没有订阅' }}
        </h2>
        <NButton v-if="data.meta.has_any_subscriptions" type="primary" @click="clearFilters"
          >清除筛选</NButton
        ><NButton v-else type="primary" @click="edit(null)"
          ><template #icon><Plus class="size-5" /></template>新增订阅</NButton
        >
      </div>
      <div
        v-if="data && data.meta.total_pages > 1"
        class="mt-8 flex items-center justify-center gap-2"
      >
        <NButton
          :disabled="data.meta.page <= 1 || status === 'pending'"
          @click="setQuery({ page: String(data.meta.page - 1) })"
          >上一页</NButton
        >
        <span class="text-sm text-muted">{{ data.meta.page }} / {{ data.meta.total_pages }}</span>
        <NButton
          :disabled="data.meta.page >= data.meta.total_pages || status === 'pending'"
          @click="setQuery({ page: String(data.meta.page + 1) })"
          >下一页</NButton
        >
      </div>
      <SubscriptionFormModal
        v-model:open="editorOpen"
        :subscription="editing"
        :default-reminder-days="data?.meta.default_reminder_days ?? 7"
        :timezone="data?.meta.timezone"
        @saved="saved"
      />
      <SubscriptionActionModal
        :open="actionOpen"
        :subscription="actionItem"
        :action="action"
        @close="closeAction"
        @saved="saved"
      />
      <SettingsModal v-model:open="settingsOpen" @saved="saved" />
      <PasswordModal v-model:open="passwordOpen" />
      <ProfileModal v-model:open="profileOpen" />
    </div>
  </main>
</template>

<style scoped>
.filter-choice {
  display: flex;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ui-border);
  padding: 0 12px;
  font-size: 14px;
  cursor: pointer;
}
.filter-choice:hover {
  border-color: var(--ui-primary);
}
input:checked + .filter-choice {
  border-color: var(--ui-primary);
  background: color-mix(in srgb, var(--ui-primary) 8%, var(--ui-bg-elevated));
  color: var(--ui-primary);
  font-weight: 600;
}
input:focus-visible + .filter-choice {
  outline: 2px solid var(--ui-primary);
  outline-offset: 2px;
}
.filter-status label + label {
  margin-left: -1px;
}
.filter-status label:first-child .filter-choice {
  border-radius: 10px 0 0 10px;
}
.filter-status label:last-child .filter-choice {
  border-radius: 0 10px 10px 0;
}
.filter-status input:checked + .filter-choice {
  position: relative;
}
@media (max-width: 767px), (pointer: coarse) {
  .filter-choice {
    min-height: 44px;
  }
}
</style>
