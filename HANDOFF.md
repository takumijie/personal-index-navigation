# 加载页首屏主题修复与音效移除交接

日期：2026-11-23

目标分支：`dev`

## 最终需求与结论

1. 后台选择经典暗色或米白后，加载页从首帧起直接使用对应主题，远程冷访问不再先暗后米白。
2. 加载页 AUDIO 功能已按需求完整移除，包括访客按钮、Web Audio 合成与解锁逻辑、后台开关及设置字段。
3. 首页昼夜主题、加载动画、音乐播放器及其他后台设置不在本次变更范围内。

## 实现说明

### 首屏主题直出

- 路由启动时读取一次站点设置，以嵌入式 `index.html` 中现有的 `<html lang="zh-CN">` 为注入点。
- 启动时预生成暗色、米白两种 HTML，同时预生成原始字节和 gzip 字节。
- HTML 注入 `data-loader-theme` 和对应内联背景色；前端 `useHomeData` 直接读取该主题作为初始设置。
- 当前响应通过 `atomic.Pointer` 选择。后台设置保存成功后仅切换指针，不在请求期间重建页面。
- `App.vue` 在路由组件挂载后移除内联背景，避免异步页面 chunk 加载期间重新露出全局暗色背景。

### 音效移除

- 删除加载页 AUDIO 按钮、声音播放、交互解锁监听及专用样式。
- 删除后台“加载页音效”开关和前后端 `sound_enabled` / `SoundEnabled` 字段。
- 旧数据库 JSON 中残留的 `sound_enabled` 会被正常忽略，无需数据库迁移。
- 首屏缓存由四种主题/音效组合简化为两种主题组合。
- 音乐播放器功能保留，与本次移除的加载页音效无关。

## 关键文件

- `internal/handler/router.go`：首屏 HTML 预生成、gzip、原子切换和响应处理。
- `internal/handler/admin.go`：设置保存后切换当前首屏响应。
- `internal/handler/public.go`：API 持有首屏缓存。
- `internal/model/models.go`、`internal/database/seed.go`：移除加载页音效设置字段。
- `frontend/src/composables/useHomeData.ts`：读取服务端直出主题。
- `frontend/src/App.vue`：挂载后清理首屏内联背景。
- `frontend/src/components/boot/BootLoader.vue`：移除 AUDIO 功能。
- `frontend/src/components/admin/SettingsPanel.vue`：移除加载页音效开关。
- `web/dist/index.html`：更新生产入口资源 hash；其余生产资源按 Git 忽略规则保留在本地构建目录。

## 性能与兼容性约束

- 不增加首屏 API 请求或串行等待，保留原有 `/api/v1/settings` 请求流程。
- 普通 HTML 请求不查询 SQLite，不在请求内生成 gzip。
- 保留 `Cache-Control: no-cache`、`Vary: Accept-Encoding`、HEAD、gzip、原始 Range 和 SPA fallback 行为。
- 带 hash 的静态资源继续走现有缓存和预压缩路径。
- 已清理 `web/dist` 历史旧 bundle，避免废弃音效代码进入 Go 嵌入资源。

## 验证结果

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `npm test --prefix frontend`：18 个测试文件、152 项测试通过。
- `npm run build --prefix frontend`：通过。
- `git diff --check`：通过。
- 隔离实例验证过首屏仅有一次 settings 请求，数据库关闭后 HTML 仍可正常从内存返回。

## 构建与发布

Windows PowerShell 中执行：

```powershell
npm test --prefix frontend
npm run build --prefix frontend
Copy-Item -Path "frontend\dist\*" -Destination "web\dist" -Recurse -Force
go test ./...
go build ./cmd/server
```

生产资源已同步到 `web/dist`。Go 服务使用嵌入资源，部署或本地验证前必须重新构建并重启服务；浏览器需强制刷新以获取新入口 HTML。

## 后续注意事项

- 加载页不再提供任何声音或 AUDIO 控件，也不再需要浏览器音频自动播放授权。
- 修改加载页主题后，后台保存成功即可切换后续 HTML 响应；已经打开的标签页仍以刷新后的响应为准。
- 若新增主题，需要同步扩展后端预生成主题列表、背景色映射和前端主题类型。
