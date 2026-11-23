// @vitest-environment happy-dom

import { beforeEach, describe, expect, it } from 'vitest'
import { DEFAULT_FAVICON, updateFavicon } from './favicon'

describe('updateFavicon', () => {
  beforeEach(() => { document.head.innerHTML = '' })

  it('使用默认图标并复用同一个 link 元素更新自定义图标', () => {
    updateFavicon('')
    const link = document.head.querySelector('link[rel~="icon"]')
    expect(link?.getAttribute('href')).toBe(DEFAULT_FAVICON)

    updateFavicon('/storage/images/custom.ico')
    expect(document.head.querySelectorAll('link[rel~="icon"]')).toHaveLength(1)
    expect(link?.getAttribute('href')).toBe('/storage/images/custom.ico')
  })

  it('已有站点图标时直接更新，不新增重复元素', () => {
    document.head.innerHTML = '<link rel="icon" href="/favicon.svg">'
    updateFavicon('https://example.com/icon.png')
    expect(document.head.querySelectorAll('link[rel~="icon"]')).toHaveLength(1)
    expect(document.head.querySelector('link')?.getAttribute('href')).toBe('https://example.com/icon.png')
  })
})
