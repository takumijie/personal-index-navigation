import type { Settings } from '@/types'

export const DEFAULT_DAY_BACKGROUND = '/images/background-day.webp'
export const DEFAULT_NIGHT_BACKGROUND = '/images/background-night.webp'
export const DEFAULT_AVATAR = '/images/avatar.webp'

export const DEFAULT_WATERMARK_TITLE = 'PERSONAL SPACE'
export const DEFAULT_WATERMARK_SUBTITLE = 'ARCHIVE // CYBERSPACE NODE'
export const DEFAULT_WATERMARK_CAPTION = 'Connected to Personal Cyberspace'

export const DEFAULT_SETTINGS: Settings = {
  site_title: 'Personal Space',
  timezone: 'Asia/Shanghai',
  time_format: '24',
  clock_label: 'LOCAL TIME',
  background: {
    day_url: DEFAULT_DAY_BACKGROUND,
    night_url: DEFAULT_NIGHT_BACKGROUND,
    day_start: '06:00',
    night_start: '18:00',
    timezone: 'Asia/Shanghai',
    duration_ms: 1200,
  },
  cards: { profile: true, navigation: true, devices: true, terminal: true, clock: true },
  loader: { theme: 'dark', sound_enabled: false, watermark_title: '', watermark_subtitle: '', watermark_caption: '', badge_image: '' },
  terminal: { fastfetch_loading_ms: 2000 },
  guestbook_auto_publish: false,
  guestbook_max_length: 280,
}
