import { onBeforeUnmount, onMounted, watch } from 'vue'
import { nextBusinessMidnight, todayInTimezone } from '@/utils/dates'

export function useBusinessDateRefresh(
  context: () => { timezone: string; today: string } | undefined,
  refresh: () => Promise<unknown>,
) {
  let mounted = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let checkedDay = ''

  function businessDay() {
    const meta = context()
    return meta ? `${meta.timezone}/${todayInTimezone(Date.now(), meta.timezone)}` : ''
  }

  function schedule() {
    clearTimeout(timer)
    const zone = context()?.timezone
    if (!mounted || !zone) return
    const now = Date.now()
    timer = setTimeout(update, nextBusinessMidnight(now, zone) - now)
  }
  async function update() {
    checkedDay = businessDay()
    try {
      await refresh()
    } catch {
      // The list request exposes its own error state; retry on the next visibility change or day.
    } finally {
      schedule()
    }
  }
  function visible() {
    if (document.visibilityState === 'visible') void update()
  }
  function received() {
    schedule()
    const meta = context()
    const currentDay = businessDay()
    // A response formed before midnight can arrive after it; compensate once per business day.
    if (
      mounted &&
      meta &&
      `${meta.timezone}/${meta.today}` !== currentDay &&
      checkedDay !== currentDay
    )
      void update()
  }
  watch(context, received)
  onMounted(() => {
    mounted = true
    received()
    document.addEventListener('visibilitychange', visible)
  })
  onBeforeUnmount(() => {
    mounted = false
    clearTimeout(timer)
    document.removeEventListener('visibilitychange', visible)
  })
}
