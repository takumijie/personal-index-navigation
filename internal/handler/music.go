package handler

import (
	"net/http"

	"homepage/internal/model"

	"github.com/gin-gonic/gin"
)

func (a *API) Playlists(c *gin.Context) {
	items, err := a.content.Playlists(true)
	if err != nil {
		handleError(c, err)
		return
	}
	result := make([]publicPlaylist, 0, len(items))
	for _, item := range items {
		result = append(result, newPublicPlaylist(item))
	}
	success(c, result)
}

func (a *API) PublicPlaylistByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := a.content.PlaylistByID(id, true)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, newPublicPlaylist(item))
}

func (a *API) Playback(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if a.music == nil {
		fail(c, http.StatusServiceUnavailable, "music_unavailable", "音乐服务尚未配置")
		return
	}
	if !a.music.AllowPlayback(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, "rate_limited", "播放接口请求过于频繁，请稍后重试")
		return
	}
	item, err := a.music.Playback(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, item)
}

func (a *API) Lyrics(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if a.music == nil {
		fail(c, http.StatusServiceUnavailable, "music_unavailable", "音乐服务尚未配置")
		return
	}
	if !a.music.AllowLyrics(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, "rate_limited", "歌词接口请求过于频繁，请稍后重试")
		return
	}
	item, err := a.music.Lyrics(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, item)
}

func (a *API) Cover(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if a.music == nil {
		fail(c, http.StatusServiceUnavailable, "music_unavailable", "音乐服务尚未配置")
		return
	}
	if !a.music.AllowCover(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, "rate_limited", "封面接口请求过于频繁，请稍后重试")
		return
	}
	item, err := a.music.Cover(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	if item.URL != "" {
		c.Redirect(http.StatusFound, item.URL)
		return
	}
	c.Data(http.StatusOK, item.ContentType, item.Data)
}

func (a *API) AdminPlaylists(c *gin.Context) {
	items, err := a.content.Playlists(false)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) CreatePlaylist(c *gin.Context) {
	var item model.Playlist
	if !bindJSON(c, &item) {
		return
	}
	created, err := a.content.CreatePlaylist(item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, created)
}

func (a *API) UpdatePlaylist(c *gin.Context) {
	var item model.Playlist
	if !bindJSON(c, &item) {
		return
	}
	if item.ID == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "播放列表 ID 无效")
		return
	}
	updated, err := a.content.UpdatePlaylist(item.ID, item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) DeletePlaylist(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if !bindJSON(c, &input) {
		return
	}
	if input.ID == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "播放列表 ID 无效")
		return
	}
	if err := a.content.DeletePlaylist(input.ID); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (a *API) SetPlaylistTracks(c *gin.Context) {
	var input struct {
		PlaylistID uint   `json:"playlist_id"`
		TrackIDs   []uint `json:"track_ids"`
	}
	if !bindJSON(c, &input) {
		return
	}
	if input.PlaylistID == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "播放列表 ID 无效")
		return
	}
	item, err := a.content.SetPlaylistTracks(input.PlaylistID, input.TrackIDs)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, item)
}
