export interface Profile {
  name: string
  username: string
  description: string
  avatar: string
  cover_mode: 'day' | 'night' | 'current' | 'custom'
  cover_url: string
  tags: string[]
}

export interface NavigationItem {
  id: number
  title: string
  url: string
  icon: string
  icon_url: string
  sort: number
  enabled: boolean
  open_mode: 'blank' | 'self'
}

export interface Device {
  id: number
  name: string
  description: string
  icon: string
  category: string
  sort: number
  enabled: boolean
}

export interface BackgroundSettings {
  day_url: string
  night_url: string
  day_start: string
  night_start: string
  timezone: string
  duration_ms: number
}

export interface CardSettings {
  profile: boolean
  navigation: boolean
  devices: boolean
  terminal: boolean
  clock: boolean
}

export interface LoaderSettings {
  watermark_title: string
  watermark_subtitle: string
  watermark_caption: string
  badge_image: string
}

export interface Settings {
  site_title: string
  timezone: string
  time_format: '12' | '24'
  clock_label: string
  background: BackgroundSettings
  cards: CardSettings
  loader: LoaderSettings
  guestbook_auto_publish: boolean
  guestbook_max_length: number
}

export interface Track {
  id: number
  title: string
  artist: string
  album: string
  cover: string
  url: string
  source: string
  sort: number
  enabled: boolean
}

export interface GuestbookEntry {
  id: number
  system_name: string
  distribution: string
  browser_name: string
  browser_version: string
  ip_address: string
  message: string
  status: 'pending' | 'published' | 'rejected'
  created_at: string
}

export interface Visitor {
  ip: string
  visit_time: string
}

export interface SystemInfo {
  os: string
  distribution: string
  distributionKey: string
  browserName: string
  browserVersion: string
}

export interface ApiErrorBody {
  code: string | number
  message: string
}
