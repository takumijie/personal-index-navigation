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
  theme: 'dark' | 'beige'
  sound_enabled: boolean
  watermark_title: string
  watermark_subtitle: string
  watermark_caption: string
  badge_image: string
}

export interface TerminalSettings {
  fastfetch_loading_ms: number
}

export interface MusicSettings {
  lyrics_enabled: boolean
  playlist_id: number
}

export interface Settings {
  version: number
  site_title: string
  timezone: string
  time_format: '12' | '24'
  clock_label: string
  background: BackgroundSettings
  cards: CardSettings
  loader: LoaderSettings
  terminal: TerminalSettings
  music: MusicSettings
  guestbook_auto_publish: boolean
  guestbook_max_length: number
}

export type TrackSource = 'local' | 'netease' | 's3' | 'remote'

export interface Track {
  id: number
  title: string
  artist: string
  album: string
  cover: string
  url: string
  source: TrackSource
  external_id: string
  object_key: string
  lyric_path: string
  sort: number
  enabled: boolean
  playlist_ids?: number[]
}

export interface PublicTrack {
  id: number
  title: string
  artist: string
  album: string
  cover: string
  source: TrackSource
  sort: number
  enabled: boolean
}

export interface Playlist {
  id: number
  name: string
  sort: number
  enabled: boolean
  tracks: Track[]
}

export interface PublicPlaylist {
  id: number
  name: string
  sort: number
  enabled: boolean
  tracks: PublicTrack[]
}

export interface Playback {
  url: string
  expires_at?: string
}

export interface Lyrics {
  text: string
  format: 'lrc'
}

export interface NeteaseConfig {
  base_url: string
}

export interface NeteaseStatus {
  authenticated: boolean
  nickname?: string
  avatar?: string
}

export interface NeteaseQrSession {
  session_id: string
  qr_image: string
  expires_at: string
}

export interface NeteaseQrState {
  status: 'waiting' | 'scanned' | 'success' | 'expired'
  message: string
}

export interface NeteaseSearchResult {
  tracks: Track[]
  total: number
  page: number
  page_size: number
}

export interface ImportResult {
  playlist: Playlist
  imported: number
  duplicates: number
}

export interface S3Config {
  endpoint: string
  region: string
  bucket: string
  access_key_id: string
  secret_access_key?: string
  clear_secret?: boolean
  prefix: string
  force_path_style: boolean
  public_base_url: string
  access_mode: 'public' | 'presigned'
  has_secret?: boolean
}

export interface S3Object {
  key: string
  size: number
  last_modified?: string
}

export interface S3ObjectsResult {
  objects: S3Object[]
  next_token: string
  has_more: boolean
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
  ip_address: string
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
