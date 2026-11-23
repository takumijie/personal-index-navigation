package handler

import (
	"net/http"
	"strconv"

	"homepage/internal/model"
	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

type API struct {
	content      *service.ContentService
	auth         *service.AuthService
	uploads      *service.UploadService
	music        *service.MusicService
	netease      *service.NeteaseService
	s3           *service.S3Service
	cookieSecure bool
}

func NewAPI(content *service.ContentService, auth *service.AuthService, uploads *service.UploadService, music *service.MusicService) *API {
	return &API{content: content, auth: auth, uploads: uploads, music: music}
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

type publicTrack struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Album   string `json:"album"`
	Cover   string `json:"cover"`
	Source  string `json:"source"`
	Sort    int    `json:"sort"`
	Enabled bool   `json:"enabled"`
}

type publicPlaylist struct {
	ID      uint          `json:"id"`
	Name    string        `json:"name"`
	Sort    int           `json:"sort"`
	Enabled bool          `json:"enabled"`
	Tracks  []publicTrack `json:"tracks"`
}

func newPublicTrack(track model.MusicTrack) publicTrack {
	return publicTrack{
		ID: track.ID, Title: track.Title, Artist: track.Artist, Album: track.Album,
		Cover: track.Cover, Source: track.Source, Sort: track.Sort, Enabled: track.Enabled,
	}
}

func newPublicTracks(tracks []model.MusicTrack) []publicTrack {
	items := make([]publicTrack, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, newPublicTrack(track))
	}
	return items
}

func newPublicPlaylist(playlist model.Playlist) publicPlaylist {
	return publicPlaylist{
		ID: playlist.ID, Name: playlist.Name, Sort: playlist.Sort, Enabled: playlist.Enabled,
		Tracks: newPublicTracks(playlist.Tracks),
	}
}

func (a *API) Tracks(c *gin.Context) {
	playlists, err := a.content.Playlists(true)
	if err != nil {
		handleError(c, err)
		return
	}
	items := make([]publicTrack, 0)
	seen := make(map[uint]struct{})
	for _, playlist := range playlists {
		for _, track := range playlist.Tracks {
			if _, ok := seen[track.ID]; ok {
				continue
			}
			seen[track.ID] = struct{}{}
			items = append(items, newPublicTrack(track))
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
	success(c, newPublicTracks(playlist.Tracks))
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
