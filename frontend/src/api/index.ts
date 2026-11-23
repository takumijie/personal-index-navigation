import { request, SLOW_REQUEST_TIMEOUT_MS, type RequestOptions } from './client'
import type {
  Device, GuestbookEntry, ImportResult, Lyrics, NavigationItem, NeteaseConfig, NeteaseQrSession, NeteaseQrState,
  NeteaseSearchResult, NeteaseStatus, Playback, Playlist, Profile, PublicPlaylist, PublicTrack, S3Config, S3ObjectsResult, Settings, Track, Visitor,
} from '@/types'

const json = (method: string, body?: unknown): RequestInit => ({
  method,
  body: body === undefined ? undefined : JSON.stringify(body),
})
const slowJson = (method: string, body?: unknown): RequestOptions => ({
  ...json(method, body),
  timeoutMs: SLOW_REQUEST_TIMEOUT_MS,
})

export const publicApi = {
  profile: () => request<Profile>('/profile'),
  navigation: () => request<NavigationItem[]>('/navigation'),
  devices: () => request<Device[]>('/devices'),
  settings: () => request<Settings>('/settings'),
  guestbook: () => request<GuestbookEntry[]>('/guestbook'),
  playlist: () => request<PublicTrack[]>('/music/playlist'),
  playlists: () => request<PublicPlaylist[]>('/music/playlists'),
  playlistByID: (id: number) => request<PublicPlaylist>(`/music/playlists/${id}`),
  playback: (id: number) => request<Playback>(`/music/tracks/${id}/playback`),
  lyrics: (id: number) => request<Lyrics>(`/music/tracks/${id}/lyrics`),
  coverUrl: (id: number) => `/api/v1/music/tracks/${id}/cover`,
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
  tracks: () => request<Track[]>('/admin/music/tracks/list', json('POST')),
  createTrack: (data: Omit<Track, 'id'>) => request<Track>('/admin/music/tracks/create', json('POST', data)),
  updateTrack: (data: Track) => request<Track>('/admin/music/tracks/update', json('POST', data)),
  deleteTrack: (id: number) => request<null>('/admin/music/tracks/delete', json('POST', { id })),
  playlists: () => request<Playlist[]>('/admin/music/playlists/list', json('POST')),
  createPlaylist: (data: Pick<Playlist, 'name' | 'sort' | 'enabled'>) => request<Playlist>('/admin/music/playlists/create', json('POST', data)),
  updatePlaylist: (data: Pick<Playlist, 'id' | 'name' | 'sort' | 'enabled'>) => request<Playlist>('/admin/music/playlists/update', json('POST', data)),
  deletePlaylist: (id: number) => request<null>('/admin/music/playlists/delete', json('POST', { id })),
  setPlaylistTracks: (playlistID: number, trackIDs: number[]) => request<Playlist>('/admin/music/playlists/tracks', json('POST', { playlist_id: playlistID, track_ids: trackIDs })),
  neteaseConfig: () => request<NeteaseConfig>('/admin/music/netease/config/read', json('POST')),
  saveNeteaseConfig: (data: NeteaseConfig) => request<NeteaseConfig>('/admin/music/netease/config/save', json('POST', data)),
  neteaseQrCreate: () => request<NeteaseQrSession>('/admin/music/netease/qr/create', json('POST')),
  neteaseQrCheck: (sessionID: string) => request<NeteaseQrState>('/admin/music/netease/qr/check', json('POST', { session_id: sessionID })),
  neteaseStatus: () => request<NeteaseStatus>('/admin/music/netease/status', json('POST')),
  neteaseLogout: () => request<null>('/admin/music/netease/logout', json('POST')),
  neteaseSearch: (keywords: string, page: number, pageSize = 20) => request<NeteaseSearchResult>('/admin/music/netease/search', json('POST', { keywords, page, page_size: pageSize })),
  neteaseCreatePlaylist: (name: string, tracks: Track[]) => request<ImportResult>('/admin/music/netease/create-playlist', slowJson('POST', { name, tracks })),
  neteaseAppendPlaylist: (playlistID: number, tracks: Track[]) => request<ImportResult>('/admin/music/netease/append-playlist', slowJson('POST', { playlist_id: playlistID, tracks })),
  neteaseImportPlaylist: (input: string, name: string) => request<ImportResult>('/admin/music/netease/import-playlist', slowJson('POST', { input, name })),
  s3Config: () => request<S3Config>('/admin/music/s3/config/read', json('POST')),
  saveS3Config: (data: S3Config) => request<S3Config>('/admin/music/s3/config/save', json('POST', data)),
  testS3: () => request<{ connected: boolean }>('/admin/music/s3/test', json('POST')),
  s3Objects: (prefix: string, continuationToken = '') => request<S3ObjectsResult>('/admin/music/s3/objects/list', json('POST', { prefix, continuation_token: continuationToken })),
  importS3: (name: string, keys: string[]) => request<ImportResult>('/admin/music/s3/import', json('POST', { name, keys })),
  appendS3: (playlistID: number, keys: string[]) => request<ImportResult>('/admin/music/s3/append-playlist', json('POST', { playlist_id: playlistID, keys })),
  deleteS3Object: (key: string) => request<{ deleted: boolean }>('/admin/music/s3/objects/delete', json('POST', { key })),
  uploadImage: (file: File) => upload('/admin/uploads/image', file),
  uploadMusic: (file: File) => upload('/admin/uploads/music', file),
  uploadLyrics: (file: File) => upload('/admin/uploads/lyrics', file),
}

async function upload(path: string, file: File): Promise<{ url: string }> {
  const body = new FormData()
  body.append('file', file)
  return request<{ url: string }>(path, { method: 'POST', body, timeoutMs: SLOW_REQUEST_TIMEOUT_MS })
}
