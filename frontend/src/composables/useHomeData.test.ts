// @vitest-environment happy-dom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '@/api'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import { useHomeData } from './useHomeData'

beforeEach(() => {
  vi.restoreAllMocks()
  vi.stubEnv('DEV', false)
  vi.spyOn(console, 'warn').mockImplementation(() => {})
  vi.spyOn(publicApi, 'profile').mockResolvedValue({ name: 'Profile' } as never)
  vi.spyOn(publicApi, 'navigation').mockResolvedValue([])
  vi.spyOn(publicApi, 'devices').mockResolvedValue([])
  vi.spyOn(publicApi, 'settings').mockResolvedValue(structuredClone(DEFAULT_SETTINGS))
  vi.spyOn(publicApi, 'guestbook').mockResolvedValue([])
  vi.spyOn(publicApi, 'visitor').mockResolvedValue(null as never)
})

afterEach(() => {
  vi.unstubAllEnvs()
  delete document.documentElement.dataset.loaderTheme
})

describe('useHomeData loader bootstrap', () => {
  it('uses the server-selected loader theme before any API request', () => {
    document.documentElement.dataset.loaderTheme = 'beige'
    const settings = vi.mocked(publicApi.settings)

    const home = useHomeData()

    expect(home.settings.value.loader.theme).toBe('beige')
    expect(settings).not.toHaveBeenCalled()
  })
})

describe('useHomeData music isolation', () => {
  it('keeps core home data available when the playlist request fails', async () => {
    vi.spyOn(publicApi, 'playlist').mockRejectedValue(new Error('music offline'))
    const home = useHomeData()

    await home.reload()

    expect(home.profile.value?.name).toBe('Profile')
    expect(home.tracks.value).toEqual([])
    expect(home.error.value).toContain('音乐暂时不可用')
    expect(home.loading.value).toBe(false)
  })

  it('retries terminal requests after a failure', async () => {
    vi.spyOn(publicApi, 'playlist').mockResolvedValue([])
    const guestbook = vi.mocked(publicApi.guestbook)
    guestbook.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce([])
    const home = useHomeData()

    await home.loadTerminalData()
    expect(home.terminalError.value).not.toBe('')

    await home.loadTerminalData()
    expect(guestbook).toHaveBeenCalledTimes(2)
    expect(home.terminalError.value).toBe('')
  })
})
