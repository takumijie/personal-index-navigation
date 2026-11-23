// @vitest-environment happy-dom

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '@/api'
import type { Track } from '@/types'
import MusicPlayer from './MusicPlayer.vue'

const tracks: Track[] = [
  {
    id: 1, title: 'First', artist: 'Artist', album: '', cover: '', url: '', source: 'remote',
    external_id: '', object_key: '', lyric_path: '', sort: 1, enabled: true,
  },
  {
    id: 2, title: 'Second', artist: 'Artist', album: '', cover: '', url: '', source: 'remote',
    external_id: '', object_key: '', lyric_path: '', sort: 2, enabled: true,
  },
]

const loadMock = vi.fn()
const playMock = vi.fn()
const pauseMock = vi.fn()
let wrapper: VueWrapper | undefined

async function createPlayer(expiresAt = '') {
  vi.mocked(publicApi.playback).mockResolvedValue({ url: 'https://example.com/first.mp3', expires_at: expiresAt })
  wrapper = mount(MusicPlayer, {
    props: { tracks, settings: { lyrics_enabled: false, playlist_id: 1 } },
  })
  await flushPromises()
  const audio = wrapper.get('audio').element as HTMLAudioElement
  Object.defineProperty(audio, 'duration', { configurable: true, get: () => 120 })
  audio.dispatchEvent(new Event('durationchange'))
  await nextTick()
  return { wrapper, audio }
}

async function startPlayback(player: VueWrapper) {
  await player.get('button[aria-label="播放"]').trigger('click')
  await flushPromises()
}

async function seekTo(player: VueWrapper, value: number) {
  await player.get('input[aria-label="播放进度"]').setValue(String(value))
}

beforeEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
  localStorage.setItem('music-playback-mode', 'sequential')
  loadMock.mockClear()
  playMock.mockClear()
  pauseMock.mockClear()
  vi.spyOn(publicApi, 'playlists').mockResolvedValue([
    { id: 1, name: 'Playlist', sort: 1, enabled: true, tracks },
  ])
  vi.spyOn(publicApi, 'playback').mockResolvedValue({ url: 'https://example.com/first.mp3' })
  vi.spyOn(HTMLMediaElement.prototype, 'load').mockImplementation(function (this: HTMLMediaElement) {
    loadMock(this)
    this.currentTime = 0
  })
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockImplementation(function (this: HTMLMediaElement) {
    playMock(this)
    this.dispatchEvent(new Event('play'))
    return Promise.resolve()
  })
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(function (this: HTMLMediaElement) {
    pauseMock(this)
    this.dispatchEvent(new Event('pause'))
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('MusicPlayer seek recovery', () => {
  it('seeks on a near-expiry loaded URL without refreshing or reloading it', async () => {
    const nearExpiry = new Date(Date.now() + 30_000).toISOString()
    const { wrapper: player, audio } = await createPlayer(nearExpiry)
    await startPlayback(player)
    const loadCalls = loadMock.mock.calls.length

    await seekTo(player, 50)

    expect(audio.currentTime).toBe(60)
    expect(publicApi.playback).toHaveBeenCalledTimes(1)
    expect(loadMock).toHaveBeenCalledTimes(loadCalls)
    expect(playMock).toHaveBeenCalledTimes(1)
  })

  it('restores the seek target after an error refresh and retries only once', async () => {
    const { wrapper: player, audio } = await createPlayer()
    vi.mocked(publicApi.playback)
      .mockResolvedValueOnce({ url: 'https://example.com/expired.mp3' })
      .mockResolvedValueOnce({ url: 'https://example.com/renewed.mp3' })
    await startPlayback(player)
    await seekTo(player, 50)

    audio.dispatchEvent(new Event('error'))
    audio.dispatchEvent(new Event('error'))
    await flushPromises()

    expect(publicApi.playback).toHaveBeenCalledTimes(2)
    expect(audio.src).toBe('https://example.com/renewed.mp3')
    expect(audio.currentTime).toBe(0)
    expect(playMock).toHaveBeenCalledTimes(1)

    audio.dispatchEvent(new Event('loadedmetadata'))
    await flushPromises()

    expect(audio.currentTime).toBe(60)
    expect(playMock).toHaveBeenCalledTimes(2)

    audio.dispatchEvent(new Event('error'))
    await flushPromises()
    expect(publicApi.playback).toHaveBeenCalledTimes(2)
    expect(playMock).toHaveBeenCalledTimes(2)
    expect(player.text()).toContain('音频加载失败')
  })

  it('keeps the seek position across an ordinary pause and resume', async () => {
    const { wrapper: player, audio } = await createPlayer()
    await startPlayback(player)
    await seekTo(player, 25)
    const loadCalls = loadMock.mock.calls.length

    await player.get('button[aria-label="暂停"]').trigger('click')
    await player.get('button[aria-label="播放"]').trigger('click')
    await flushPromises()

    expect(audio.currentTime).toBe(30)
    expect(publicApi.playback).toHaveBeenCalledTimes(1)
    expect(loadMock).toHaveBeenCalledTimes(loadCalls)
    expect(playMock).toHaveBeenCalledTimes(2)
  })

  it('uses the latest playback position when an expired URL is refreshed on resume', async () => {
    const { wrapper: player, audio } = await createPlayer()
    vi.mocked(publicApi.playback)
      .mockResolvedValueOnce({ url: 'https://example.com/expired.mp3', expires_at: new Date(Date.now() - 1000).toISOString() })
      .mockResolvedValueOnce({ url: 'https://example.com/renewed.mp3' })
    await startPlayback(player)
    await seekTo(player, 50)
    audio.currentTime = 90
    audio.dispatchEvent(new Event('timeupdate'))

    await player.get('button[aria-label="暂停"]').trigger('click')
    await player.get('button[aria-label="播放"]').trigger('click')
    await flushPromises()
    audio.dispatchEvent(new Event('loadedmetadata'))
    await flushPromises()

    expect(publicApi.playback).toHaveBeenCalledTimes(2)
    expect(audio.currentTime).toBe(90)
  })

  it('clears a pending seek target when changing tracks', async () => {
    const { wrapper: player, audio } = await createPlayer()
    await startPlayback(player)
    await seekTo(player, 50)

    await player.get('button[aria-label="下一曲"]').trigger('click')
    await flushPromises()
    audio.dispatchEvent(new Event('loadedmetadata'))
    await flushPromises()

    expect(publicApi.playback).toHaveBeenCalledTimes(2)
    expect(player.get('h3').text()).toBe('Second')
    expect(audio.currentTime).toBe(0)
  })

  it('starts repeat-one from zero instead of restoring the previous seek', async () => {
    localStorage.setItem('music-playback-mode', 'repeat-one')
    const { wrapper: player, audio } = await createPlayer()
    await startPlayback(player)
    await seekTo(player, 50)
    const loadCalls = loadMock.mock.calls.length

    audio.dispatchEvent(new Event('ended'))
    await flushPromises()

    expect(audio.currentTime).toBe(0)
    expect(publicApi.playback).toHaveBeenCalledTimes(1)
    expect(loadMock).toHaveBeenCalledTimes(loadCalls)
    expect(playMock).toHaveBeenCalledTimes(2)
  })
})
