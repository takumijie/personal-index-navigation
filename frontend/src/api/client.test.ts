import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { adminApi } from './index'
import {
  DEFAULT_REQUEST_TIMEOUT_MS,
  request,
  SLOW_REQUEST_TIMEOUT_MS,
} from './client'

function pendingFetch() {
  const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
    const abort = () => reject(new DOMException('Aborted', 'AbortError'))
    if (init?.signal?.aborted) abort()
    else init?.signal?.addEventListener('abort', abort, { once: true })
  }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('API request cancellation', () => {
  it('returns data and clears the timeout after a normal response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { ok: true } }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })))

    await expect(request<{ ok: boolean }>('/ok')).resolves.toEqual({ ok: true })
    expect(vi.getTimerCount()).toBe(0)
  })

  it('times out a pending request with the default deadline', async () => {
    const fetchMock = pendingFetch()
    const pending = request('/slow')
    const assertion = expect(pending).rejects.toMatchObject({ code: 'TIMEOUT', status: 0 })

    await vi.advanceTimersByTimeAsync(DEFAULT_REQUEST_TIMEOUT_MS)

    await assertion
    expect(fetchMock.mock.calls[0][1]?.signal?.aborted).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('keeps the internal timeout when the caller supplies a signal', async () => {
    pendingFetch()
    const outside = new AbortController()
    const pending = request('/slow', { signal: outside.signal })
    const assertion = expect(pending).rejects.toMatchObject({ code: 'TIMEOUT' })

    await vi.advanceTimersByTimeAsync(DEFAULT_REQUEST_TIMEOUT_MS)

    await assertion
    expect(outside.signal.aborted).toBe(false)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('distinguishes caller cancellation and clears the timeout', async () => {
    const fetchMock = pendingFetch()
    const outside = new AbortController()
    const pending = request('/cancel', { signal: outside.signal })
    const assertion = expect(pending).rejects.toMatchObject({ code: 'REQUEST_ABORTED', status: 0 })

    outside.abort()

    await assertion
    expect(fetchMock.mock.calls[0][1]?.signal?.aborted).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('uses the five-minute deadline for Netease bulk operations', async () => {
    pendingFetch()
    const pending = adminApi.neteaseCreatePlaylist('Daily', [])
    const assertion = expect(pending).rejects.toMatchObject({ code: 'TIMEOUT' })

    await vi.advanceTimersByTimeAsync(DEFAULT_REQUEST_TIMEOUT_MS)
    expect(vi.getTimerCount()).toBe(1)
    await vi.advanceTimersByTimeAsync(SLOW_REQUEST_TIMEOUT_MS - DEFAULT_REQUEST_TIMEOUT_MS)

    await assertion
    expect(vi.getTimerCount()).toBe(0)
  })
})
