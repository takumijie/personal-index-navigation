import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import type { Settings } from '@/types'

function zonedMinutes(date: Date, timezone: string): number {
  try {
    const parts = new Intl.DateTimeFormat('en-GB', {
      timeZone: timezone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
    }).formatToParts(date)
    const hour = Number(parts.find((part) => part.type === 'hour')?.value || 0)
    const minute = Number(parts.find((part) => part.type === 'minute')?.value || 0)
    return hour * 60 + minute
  } catch {
    return date.getHours() * 60 + date.getMinutes()
  }
}

function parseTime(value: string): number {
  const [hour, minute] = value.split(':').map(Number)
  return (hour || 0) * 60 + (minute || 0)
}

export function isDaytime(date: Date, settings: Settings['background']): boolean {
  const current = zonedMinutes(date, settings.timezone)
  const start = parseTime(settings.day_start)
  const end = parseTime(settings.night_start)
  return start < end ? current >= start && current < end : current >= start || current < end
}

export function useBackground(settings: Ref<Settings>) {
  const now = ref(new Date())
  const loaded = ref(new Set<string>())
  const reducedMotion = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let motionQuery: MediaQueryList | undefined
  let updateMotion: (() => void) | undefined

  const isDay = computed(() => isDaytime(now.value, settings.value.background))
  const mode = computed<'day' | 'night'>(() => isDay.value ? 'day' : 'night')
  const activeUrl = computed(() => mode.value === 'day' ? settings.value.background.day_url : settings.value.background.night_url)
  const duration = computed(() => reducedMotion.value ? 0 : Math.max(0, settings.value.background.duration_ms))

  function preload(url: string) {
    if (!url || loaded.value.has(url)) return
    const image = new Image()
    image.onload = () => loaded.value.add(url)
    image.src = url
  }

  function schedule() {
    if (timer) clearTimeout(timer)
    now.value = new Date()
    timer = setTimeout(schedule, 60_000 - (Date.now() % 60_000) + 80)
  }

  watch(() => [settings.value.background.day_url, settings.value.background.night_url], ([day, night]) => {
    if (day) preload(day)
    if (night) preload(night)
  }, { immediate: true })

  onMounted(() => {
    motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    reducedMotion.value = motionQuery.matches
    updateMotion = () => { reducedMotion.value = motionQuery?.matches || false }
    motionQuery.addEventListener('change', updateMotion)
    schedule()
  })

  onBeforeUnmount(() => {
    if (timer) clearTimeout(timer)
    if (updateMotion) motionQuery?.removeEventListener('change', updateMotion)
  })
  return { mode, isDay, activeUrl, loaded, duration, preload }
}
