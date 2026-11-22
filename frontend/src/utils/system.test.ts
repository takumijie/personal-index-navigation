import { afterEach, describe, expect, it, vi } from 'vitest'
import { detectSystem } from './system'

afterEach(() => vi.unstubAllGlobals())

describe('detectSystem', () => {
  it('normalizes Linux distributions and detects Chromium-based browsers', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (X11; Arch Linux x86_64) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0',
    })

    const result = await detectSystem()

    expect(result).toMatchObject({ os: 'Linux', distributionKey: 'linux', browserName: 'Edge', browserVersion: '130.0.0.0' })
  })

  it('keeps Linux generic when high entropy platform data names a distribution', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36',
      userAgentData: {
        platform: 'Linux',
        getHighEntropyValues: vi.fn().mockResolvedValue({ platform: 'Ubuntu Linux' }),
      },
      language: 'zh-CN',
    })

    const result = await detectSystem()

    expect(result.os).toBe('Linux')
    expect(result.distributionKey).toBe('linux')
    expect(result.browserName).toBe('Chrome')
  })

  it.each([
    ['Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36', 'ChromeOS', 'chromeos'],
    ['Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36', 'FreeBSD', 'freebsd'],
  ])('distinguishes %s from generic Linux', async (userAgent, os, distributionKey) => {
    vi.stubGlobal('navigator', { userAgent })

    const result = await detectSystem()

    expect(result).toMatchObject({ os, distributionKey })
  })

  it('extracts mobile OS versions and Safari versions', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 Version/18.1 Mobile/15E148 Safari/604.1',
    })

    const result = await detectSystem()

    expect(result).toMatchObject({ os: 'iOS 18.1', distributionKey: 'ios', browserName: 'Safari', browserVersion: '18.1' })
  })
})
