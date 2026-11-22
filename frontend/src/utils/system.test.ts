import { afterEach, describe, expect, it, vi } from 'vitest'
import { detectSystem } from './system'

afterEach(() => vi.unstubAllGlobals())

describe('detectSystem', () => {
  it('detects Linux distributions and Chromium-based browsers', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (X11; Arch Linux x86_64) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0',
    })

    const result = await detectSystem()

    expect(result).toMatchObject({ os: 'Arch Linux', distributionKey: 'arch', browserName: 'Edge', browserVersion: '130.0.0.0' })
  })

  it('uses high entropy platform data when available', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36',
      userAgentData: {
        platform: 'Linux',
        getHighEntropyValues: vi.fn().mockResolvedValue({ platform: 'Ubuntu Linux' }),
      },
      language: 'zh-CN',
    })

    const result = await detectSystem()

    expect(result.os).toBe('Ubuntu')
    expect(result.distributionKey).toBe('ubuntu')
    expect(result.browserName).toBe('Chrome')
  })

  it('extracts mobile OS versions and Safari versions', async () => {
    vi.stubGlobal('navigator', {
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 Version/18.1 Mobile/15E148 Safari/604.1',
    })

    const result = await detectSystem()

    expect(result).toMatchObject({ os: 'iOS 18.1', distributionKey: 'ios', browserName: 'Safari', browserVersion: '18.1' })
  })
})
