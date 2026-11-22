const VIDEO_EXTENSION = /\.(mp4|webm|ogg|ogv|mov|m4v)$/i

/** 按扩展名判断背景资源是否为视频（忽略 query 与 hash） */
export function isVideoUrl(url: string): boolean {
  if (!url) return false
  const clean = url.split(/[?#]/, 1)[0]
  return VIDEO_EXTENSION.test(clean)
}
