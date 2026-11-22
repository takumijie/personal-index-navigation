import { DEFAULT_AVATAR, DEFAULT_DAY_BACKGROUND, DEFAULT_NIGHT_BACKGROUND, DEFAULT_SETTINGS } from '@/data/defaults'
import type { Device, GuestbookEntry, NavigationItem, Profile, Settings, Track, Visitor } from '@/types'

export const mockProfile: Profile = {
  name: '林暮',
  username: '@twilight',
  description: '在代码与暮色之间，收集安静而闪亮的瞬间。全栈开发者，偶尔拍照，持续好奇。',
  avatar: DEFAULT_AVATAR,
  cover_mode: 'current',
  cover_url: '',
  tags: ['全栈开发', '摄影', '咖啡', '开源'],
}

export const mockNavigation: NavigationItem[] = [
  { id: 1, title: '我的博客', url: 'https://example.com', icon: 'BookOpen', icon_url: '', sort: 1, enabled: true, open_mode: 'blank' },
  { id: 2, title: 'GitHub', url: 'https://github.com', icon: 'Github', icon_url: '', sort: 2, enabled: true, open_mode: 'blank' },
  { id: 3, title: '摄影作品', url: 'https://example.com/gallery', icon: 'Camera', icon_url: '', sort: 3, enabled: true, open_mode: 'blank' },
  { id: 4, title: '联系我', url: 'mailto:hello@example.com', icon: 'Mail', icon_url: '', sort: 4, enabled: true, open_mode: 'self' },
]

export const mockDevices: Device[] = [
  { id: 1, name: 'MacBook Pro', description: 'M3 Pro · 18GB', icon: 'Laptop', category: '日常主力', sort: 1, enabled: true },
  { id: 2, name: 'iPhone', description: '随手记录生活', icon: 'Smartphone', category: '移动设备', sort: 2, enabled: true },
  { id: 3, name: 'Keychron K2', description: '茶轴 · 无线机械键盘', icon: 'Keyboard', category: '桌面外设', sort: 3, enabled: true },
]

export const mockTracks: Track[] = [
  { id: 1, title: 'Evening Walk', artist: 'SoundHelix', album: 'Afterglow', cover: DEFAULT_DAY_BACKGROUND, url: 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3', source: 'remote', sort: 1, enabled: true },
  { id: 2, title: 'Midnight Signals', artist: 'SoundHelix', album: 'Afterglow', cover: DEFAULT_NIGHT_BACKGROUND, url: 'https://www.soundhelix.com/examples/mp3/SoundHelix-Song-2.mp3', source: 'remote', sort: 2, enabled: true },
]

export const mockGuestbook: GuestbookEntry[] = [
  { id: 1, system_name: 'Arch Linux', distribution: 'arch', browser_name: 'Firefox', browser_version: '133.0', ip_address: '192.168.1.18', message: '暮色很好看，愿这里一直保留这份安静。', status: 'published', created_at: new Date(Date.now() - 86400000).toISOString() },
  { id: 2, system_name: 'macOS', distribution: 'macos', browser_name: 'Safari', browser_version: '18.1', ip_address: '10.24.8.6', message: '从终端留言，是一件很酷的事。', status: 'published', created_at: new Date(Date.now() - 43200000).toISOString() },
]

export const mockVisitor: Visitor = { ip: '127.0.0.1', visit_time: new Date().toISOString() }
export const mockSettings: Settings = {
  ...DEFAULT_SETTINGS,
  site_title: '暮色的数字角落',
  clock_label: 'SHANGHAI · UTC+8',
  background: { ...DEFAULT_SETTINGS.background, day_url: DEFAULT_DAY_BACKGROUND, night_url: DEFAULT_NIGHT_BACKGROUND },
}
