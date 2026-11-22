package handler

import (
	"net/http"

	"homepage/internal/model"
	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

func (a *API) NeteaseQRCreate(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	result, err := a.netease.QRCreate(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"session_id": result.SessionID, "qr_image": result.QRCodeBase64, "expires_at": result.ExpiresAt})
}

func (a *API) NeteaseQRCheck(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	var input struct {
		SessionID string `json:"session_id"`
	}
	if !bindJSON(c, &input) {
		return
	}
	if len(input.SessionID) != 43 {
		fail(c, http.StatusBadRequest, "invalid_session", "二维码会话无效")
		return
	}
	result, err := a.netease.QRCheck(c.Request.Context(), input.SessionID)
	if err != nil {
		handleError(c, err)
		return
	}
	status := result.Status
	if status == "authenticated" {
		status = "success"
	}
	success(c, gin.H{"status": status, "message": result.Message})
}

func (a *API) NeteaseStatus(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	result, err := a.netease.Status(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	response := gin.H{"authenticated": result.Authenticated}
	if result.Profile != nil {
		response["nickname"] = result.Profile.Nickname
		response["avatar"] = result.Profile.AvatarURL
	}
	success(c, response)
}

func (a *API) NeteaseLogout(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	if err := a.netease.Logout(c.Request.Context()); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"authenticated": false})
}

func (a *API) NeteaseSearch(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	var input struct {
		Keywords string `json:"keywords"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	if !bindJSON(c, &input) {
		return
	}
	result, err := a.netease.Search(c.Request.Context(), input.Keywords, input.Page, input.PageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, result)
}

func (a *API) NeteaseCreatePlaylist(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	var input struct {
		Name   string             `json:"name"`
		Sort   int                `json:"sort"`
		Tracks []model.MusicTrack `json:"tracks"`
	}
	if !bindJSON(c, &input) {
		return
	}
	playlist, duplicates, err := a.netease.CreateFromTracks(input.Name, input.Sort, input.Tracks)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, importResult(playlist, duplicates))
}

func (a *API) NeteaseImportPlaylist(c *gin.Context) {
	if a.neteaseUnavailable(c) {
		return
	}
	var input struct {
		Input string `json:"input"`
		Name  string `json:"name"`
	}
	if !bindJSON(c, &input) {
		return
	}
	playlist, duplicates, err := a.netease.ImportPlaylist(c.Request.Context(), input.Input, input.Name)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, importResult(playlist, duplicates))
}

func (a *API) S3ReadConfig(c *gin.Context) {
	if a.s3Unavailable(c, false) {
		return
	}
	result, err := a.s3.ReadConfig()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, result)
}

func (a *API) S3SaveConfig(c *gin.Context) {
	if a.s3Unavailable(c, true) {
		return
	}
	var input service.S3Config
	if !bindJSON(c, &input) {
		return
	}
	if err := a.s3.SaveConfig(input); err != nil {
		handleError(c, err)
		return
	}
	result, err := a.s3.ReadConfig()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, result)
}

func (a *API) S3Test(c *gin.Context) {
	if a.s3Unavailable(c, true) {
		return
	}
	if err := a.s3.Test(c.Request.Context()); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"connected": true})
}

func (a *API) S3Objects(c *gin.Context) {
	if a.s3Unavailable(c, true) {
		return
	}
	var input struct {
		Prefix            string `json:"prefix"`
		ContinuationToken string `json:"continuation_token"`
	}
	if !bindJSON(c, &input) {
		return
	}
	result, err := a.s3.ListObjects(c.Request.Context(), input.Prefix, input.ContinuationToken)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, result)
}

func (a *API) S3Import(c *gin.Context) {
	if a.s3Unavailable(c, true) {
		return
	}
	var input struct {
		Name string   `json:"name"`
		Sort int      `json:"sort"`
		Keys []string `json:"keys"`
	}
	if !bindJSON(c, &input) {
		return
	}
	playlist, duplicates, err := a.s3.Import(c.Request.Context(), input.Name, input.Sort, input.Keys)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, importResult(playlist, duplicates))
}

func importResult(playlist model.Playlist, duplicates int) gin.H {
	return gin.H{"playlist": playlist, "imported": len(playlist.Tracks), "duplicates": duplicates}
}

func (a *API) neteaseUnavailable(c *gin.Context) bool {
	if a.netease == nil {
		fail(c, http.StatusServiceUnavailable, "provider_unavailable", "网易云服务尚未配置")
		return true
	}
	return a.musicRateLimited(c)
}

func (a *API) s3Unavailable(c *gin.Context, limited bool) bool {
	if a.s3 == nil {
		fail(c, http.StatusServiceUnavailable, "provider_unavailable", "S3 服务尚未配置")
		return true
	}
	return limited && a.musicRateLimited(c)
}

func (a *API) musicRateLimited(c *gin.Context) bool {
	if a.music != nil && !a.music.Allow(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, "rate_limited", "音乐接口请求过于频繁，请稍后重试")
		return true
	}
	return false
}
