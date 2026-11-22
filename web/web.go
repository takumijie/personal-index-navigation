package web

import "embed"

// Assets 包含生产构建后的前端静态资源。
//
//go:embed all:dist
var Assets embed.FS
