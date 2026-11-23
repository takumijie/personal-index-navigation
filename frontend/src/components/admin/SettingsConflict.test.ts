// @vitest-environment happy-dom

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminApi, publicApi } from '@/api'
import { ApiError } from '@/api/client'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { Settings } from '@/types'
import SettingsPanel from './SettingsPanel.vue'
import PlaylistManager from './music/PlaylistManager.vue'

function settings(version: number): Settings {
  return { ...structuredClone(DEFAULT_SETTINGS), version }
}

function button(wrapper: VueWrapper, text: string) {
  const match = wrapper.findAll('button').find((item) => item.text() === text)
  if (!match) throw new Error(`找不到按钮：${text}`)
  return match
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('设置乐观并发冲突', () => {
  it('站点设置保留编辑和最新值，仅在手势确认后基于新版本重试', async () => {
    const initial = settings(1)
    const latest = { ...settings(2), site_title: 'server latest', music: { lyrics_enabled: false, playlist_id: 7 } }
    const read = vi.spyOn(publicApi, 'settings').mockResolvedValueOnce(initial).mockResolvedValueOnce(latest)
    const update = vi.spyOn(adminApi, 'updateSettings')
      .mockRejectedValueOnce(new ApiError('版本冲突', 409, 'settings_conflict'))
      .mockImplementationOnce(async (data) => ({ ...data, version: data.version + 1 }))
    const wrapper = mount(SettingsPanel)
    await flushPromises()

    await wrapper.get('input[maxlength="100"]').setValue('my edit')
    await button(wrapper, '保存设置').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(1)
    expect(update.mock.calls[0][0]).toMatchObject({ version: 1, site_title: 'my edit' })
    expect(read).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-testid="settings-conflict"]').text()).toContain('server latest')
    expect((wrapper.get('input[maxlength="100"]').element as HTMLInputElement).value).toBe('my edit')

    await button(wrapper, '保留当前编辑并重试').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(2)
    expect(update.mock.calls[1][0]).toMatchObject({
      version: 2,
      site_title: 'my edit',
      music: { lyrics_enabled: false, playlist_id: 7 },
    })
    expect(wrapper.find('[data-testid="settings-conflict"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('播放设置冲突后保留列表编辑，并在显式重试时采用其他面板最新字段', async () => {
    const initial = settings(1)
    const latest = { ...settings(2), site_title: 'settings panel latest' }
    vi.spyOn(publicApi, 'settings').mockResolvedValueOnce(initial).mockResolvedValueOnce(latest)
    vi.spyOn(adminApi, 'playlists').mockResolvedValue([{ id: 1, name: 'Daily', sort: 0, enabled: true, tracks: [] }])
    const update = vi.spyOn(adminApi, 'updateSettings')
      .mockRejectedValueOnce(new ApiError('版本冲突', 409, 'settings_conflict'))
      .mockImplementationOnce(async (data) => ({ ...data, version: data.version + 1 }))
    const wrapper = mount(PlaylistManager)
    await flushPromises()

    await wrapper.get('select').setValue('1')
    await button(wrapper, '保存设置').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="settings-conflict"]').text()).toContain('settings panel latest')
    expect((wrapper.get('select').element as HTMLSelectElement).value).toBe('1')

    await button(wrapper, '保留当前编辑并重试').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(2)
    expect(update.mock.calls[1][0]).toMatchObject({
      version: 2,
      site_title: 'settings panel latest',
      music: { playlist_id: 1 },
    })
    expect(wrapper.find('[data-testid="settings-conflict"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
