import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'
import type { Settings } from '@/types'

export function useClock(settings: Ref<Settings>) {
  const now = ref(new Date())
  let timer: ReturnType<typeof setInterval> | undefined
  onMounted(() => { timer = setInterval(() => { now.value = new Date() }, 250) })
  onBeforeUnmount(() => { if (timer) clearInterval(timer) })

  const zonedTime = computed(() => {
    try {
      return new Intl.DateTimeFormat('zh-CN', {
        timeZone: settings.value.timezone,
        hour: '2-digit', minute: '2-digit', second: '2-digit',
        hourCycle: settings.value.time_format === '12' ? 'h12' : 'h23',
      }).format(now.value)
    } catch {
      return now.value.toLocaleTimeString('zh-CN')
    }
  })

  const zonedDate = computed(() => {
    try {
      return new Intl.DateTimeFormat('zh-CN', {
        timeZone: settings.value.timezone, month: 'short', day: '2-digit', weekday: 'short',
      }).format(now.value)
    } catch {
      return now.value.toLocaleDateString('zh-CN')
    }
  })

  const hands = computed(() => {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: settings.value.timezone, hour: 'numeric', minute: 'numeric', second: 'numeric', hourCycle: 'h23',
    }).formatToParts(now.value)
    const value = (type: string) => Number(parts.find((part) => part.type === type)?.value || 0)
    const seconds = value('second') + now.value.getMilliseconds() / 1000
    const minutes = value('minute') + seconds / 60
    const hours = value('hour') % 12 + minutes / 60
    return { hour: hours * 30, minute: minutes * 6, second: seconds * 6 }
  })

  return { now, zonedTime, zonedDate, hands }
}
