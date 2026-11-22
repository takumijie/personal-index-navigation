export interface HalftoneOptions {
  /** 采样单元格边长（输出坐标系像素），越大点阵越稀疏 */
  cellSize?: number
  /** 输出最大宽度（像素），保持原图宽高比 */
  maxWidth?: number
  /** 圆点颜色 */
  color?: string
}

/**
 * 将图片转换为半调点阵 canvas：按网格采样亮度，映射为不同半径的圆点。
 * 透明底，一次性转换。跨域图片读取像素失败时抛出异常，由调用方降级处理。
 */
export function imageToHalftone(image: HTMLImageElement, options: HalftoneOptions = {}): HTMLCanvasElement {
  const { cellSize = 4, maxWidth = 160, color = '#e6e3d3' } = options
  const sourceWidth = image.naturalWidth
  const sourceHeight = image.naturalHeight
  if (!sourceWidth || !sourceHeight) throw new Error('图片尚未加载完成')

  const scale = Math.min(1, maxWidth / sourceWidth)
  const width = Math.max(cellSize, Math.round(sourceWidth * scale))
  const height = Math.max(cellSize, Math.round(sourceHeight * scale))

  const sampler = document.createElement('canvas')
  sampler.width = width
  sampler.height = height
  const samplerContext = sampler.getContext('2d', { willReadFrequently: true })
  if (!samplerContext) throw new Error('无法创建采样画布')
  samplerContext.drawImage(image, 0, 0, width, height)
  const pixels = samplerContext.getImageData(0, 0, width, height).data

  const output = document.createElement('canvas')
  output.width = width
  output.height = height
  const context = output.getContext('2d')
  if (!context) throw new Error('无法创建点阵画布')
  context.fillStyle = color

  for (let y = 0; y < height; y += cellSize) {
    for (let x = 0; x < width; x += cellSize) {
      let luminance = 0
      let weight = 0
      for (let dy = 0; dy < cellSize && y + dy < height; dy++) {
        for (let dx = 0; dx < cellSize && x + dx < width; dx++) {
          const index = ((y + dy) * width + (x + dx)) * 4
          const alpha = pixels[index + 3] / 255
          if (alpha === 0) continue
          // Rec.601 亮度，透明度作为采样权重
          const gray = 0.299 * pixels[index] + 0.587 * pixels[index + 1] + 0.114 * pixels[index + 2]
          luminance += gray * alpha
          weight += alpha
        }
      }
      if (weight === 0) continue
      const brightness = luminance / weight / 255
      const radius = (cellSize / 2) * Math.sqrt(brightness)
      if (radius < 0.3) continue
      context.beginPath()
      context.arc(x + cellSize / 2, y + cellSize / 2, Math.min(radius, cellSize / 2), 0, Math.PI * 2)
      context.fill()
    }
  }
  return output
}
