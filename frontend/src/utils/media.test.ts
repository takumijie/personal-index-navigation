import { describe, expect, it } from 'vitest'
import { isVideoUrl } from './media'

describe('isVideoUrl', () => {
  it('识别常见视频扩展名', () => {
    expect(isVideoUrl('https://example.com/bg.mp4')).toBe(true)
    expect(isVideoUrl('/storage/videos/night.webm')).toBe(true)
    expect(isVideoUrl('https://example.com/a.mov')).toBe(true)
    expect(isVideoUrl('https://example.com/a.m4v')).toBe(true)
    expect(isVideoUrl('https://example.com/a.ogv')).toBe(true)
  })

  it('忽略 query 与 hash', () => {
    expect(isVideoUrl('https://example.com/bg.mp4?token=abc&size=1920')).toBe(true)
    expect(isVideoUrl('https://example.com/bg.webm#t=10')).toBe(true)
  })

  it('扩展名大小写不敏感', () => {
    expect(isVideoUrl('https://example.com/BG.MP4')).toBe(true)
    expect(isVideoUrl('https://example.com/bg.WebM')).toBe(true)
  })

  it('图片与其他地址返回 false', () => {
    expect(isVideoUrl('/images/background-day.webp')).toBe(false)
    expect(isVideoUrl('https://example.com/photo.jpg?x=.mp4')).toBe(false)
    expect(isVideoUrl('https://example.com/video')).toBe(false)
    expect(isVideoUrl('https://example.com/mp4')).toBe(false)
    expect(isVideoUrl('')).toBe(false)
  })
})
