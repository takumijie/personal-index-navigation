import { describe, expect, it } from 'vitest'
import { playbackNeedsRefresh } from './playback'

describe('playbackNeedsRefresh', () => {
  const now = Date.parse('2026-01-01T00:00:00.000Z')

  it('keeps public and local URLs without an expiry reusable', () => {
    expect(playbackNeedsRefresh('', now)).toBe(false)
  })

  it('refreshes expired, invalid, and near-expiry URLs', () => {
    expect(playbackNeedsRefresh('2025-12-31T23:59:00.000Z', now)).toBe(true)
    expect(playbackNeedsRefresh('2026-01-01T00:00:30.000Z', now)).toBe(true)
    expect(playbackNeedsRefresh('not-a-date', now)).toBe(true)
  })

  it('reuses URLs outside the sixty-second safety window', () => {
    expect(playbackNeedsRefresh('2026-01-01T00:01:01.000Z', now)).toBe(false)
  })
})
