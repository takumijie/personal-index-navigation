package handler

import (
	"net/http"
	"strconv"

	"homepage/internal/model"
	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

type API struct {
	content *service.ContentService
	auth    *service.AuthService
	uploads *service.UploadService
	music   *service.MusicService
	netease *service.NeteaseService
	s3      *service.S3Service
}

func NewAPI(content *service.ContentService, auth *service.AuthService, uploads *service.UploadService, music ...*service.MusicService) *API {
	api := &API{content: content, auth: auth, uploads: uploads}
	if len(music) > 0 {
		api.music = music[0]
	}
	return api
}

func (a *API) SetMusicProviders(netease *service.NeteaseService, s3 *service.S3Service) {
	a.netease = netease
	a.s3 = s3
}

func (a *API) Health(c *gin.Context) {
	success(c, gin.H{"status": "ok"})
}

func (a *API) Profile(c *gin.Context) {
	profile, err := a.content.Profile()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, profile)
}

func (a *API) Navigation(c *gin.Context) {
	items, err := a.content.Navigation(true)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) Devices(c *gin.Context) {
	items, err := a.content.Devices(true)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) Settings(c *gin.Context) {
	settings, err := a.content.Settings()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, settings)
}

func (a *API) Guestbook(c *gin.Context) {
	limit := 50
	if value := c.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			fail(c, http.StatusBadRequest, "invalid_limit", "limit 必须是 1 到 100 的整数")
			return
		}
		limit = parsed
	}
	items, err := a.content.Guestbook(limit)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) CreateGuestbook(c *gin.Context) {
	var input service.GuestbookInput
	if !bindJSON(c, &input) {
		return
	}
	entry, err := a.content.CreateGuestbook(input, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, entry)
}

func (a *API) Tracks(c *gin.Context) {
	playlists, err := a.content.Playlists(true)
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]model.MusicTrack, 0)
	seen := make(map[uint]struct{})
	for _, playlist := range playlists {
		for _, track := range playlist.Tracks {
			if _, ok := seen[track.ID]; ok {
				continue
			}
			seen[track.ID] = struct{}{}
			items = append(items, track)
		}
	}
	success(c, items)
}

func (a *API) Playlist(c *gin.Context) {
	playlist, err := a.content.Playlist()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, playlist.Tracks)
}

func (a *API) Visitor(c *gin.Context) {
	var input service.GuestbookInput
	if c.Request.Method == http.MethodPost && !bindJSON(c, &input) {
		return
	}
	visitor, err := a.content.Visitor(input, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, visitor)
}
