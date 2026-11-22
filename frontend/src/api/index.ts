import { request } from './client'
import type { Device, GuestbookEntry, NavigationItem, Profile, Settings, Track, Visitor } from '@/types'

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  body: body === undefined ? undefined : JSON.stringify(body),
})

export const publicApi = {
  profile: () => request<Profile>('/profile'),
  navigation: () => request<NavigationItem[]>('/navigation'),
  devices: () => request<Device[]>('/devices'),
  settings: () => request<Settings>('/settings'),
  guestbook: () => request<GuestbookEntry[]>('/guestbook'),
  playlist: () => request<Track[]>('/music/playlist'),
  visitor: () => request<Visitor>('/visitor'),
  submitGuestbook: (body: Pick<GuestbookEntry, 'message' | 'system_name' | 'distribution' | 'browser_name' | 'browser_version'>) =>
    request<GuestbookEntry>('/guestbook', json('POST', body)),
}

export const adminApi = {
  login: (username: string, password: string) => request<{ username: string }>('/admin/login', json('POST', { username, password })),
  logout: () => request<null>('/admin/logout', json('POST')),
  session: () => request<{ username: string; authenticated?: boolean }>('/admin/session'),
  updateProfile: (data: Profile) => request<Profile>('/admin/profile', json('PUT', data)),
  navigation: () => request<NavigationItem[]>('/admin/navigation'),
  devices: () => request<Device[]>('/admin/devices'),
  createNavigation: (data: Omit<NavigationItem, 'id'>) => request<NavigationItem>('/admin/navigation', json('POST', data)),
  updateNavigation: (data: NavigationItem) => request<NavigationItem>(`/admin/navigation/${data.id}`, json('PUT', data)),
  deleteNavigation: (id: number) => request<null>(`/admin/navigation/${id}`, json('DELETE')),
  createDevice: (data: Omit<Device, 'id'>) => request<Device>('/admin/devices', json('POST', data)),
  updateDevice: (data: Device) => request<Device>(`/admin/devices/${data.id}`, json('PUT', data)),
  deleteDevice: (id: number) => request<null>(`/admin/devices/${id}`, json('DELETE')),
  guestbook: () => request<GuestbookEntry[]>('/admin/guestbook'),
  setGuestbookStatus: (id: number, status: GuestbookEntry['status']) => request<GuestbookEntry>(`/admin/guestbook/${id}`, json('PATCH', { status })),
  deleteGuestbook: (id: number) => request<null>(`/admin/guestbook/${id}`, json('DELETE')),
  updateSettings: (data: Settings) => request<Settings>('/admin/settings', json('PUT', data)),
  tracks: () => request<Track[]>('/admin/music/tracks'),
  createTrack: (data: Omit<Track, 'id'>) => request<Track>('/admin/music/tracks', json('POST', data)),
  updateTrack: (data: Track) => request<Track>(`/admin/music/tracks/${data.id}`, json('PUT', data)),
  deleteTrack: (id: number) => request<null>(`/admin/music/tracks/${id}`, json('DELETE')),
  uploadImage: (file: File) => upload('/admin/uploads/image', file),
  uploadMusic: (file: File) => upload('/admin/uploads/music', file),
}

async function upload(path: string, file: File): Promise<{ url: string }> {
  const body = new FormData()
  body.append('file', file)
  return request<{ url: string }>(path, { method: 'POST', body })
}
