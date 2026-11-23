package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"homepage/internal/model"

	"github.com/gin-gonic/gin"
)

func TestPublicPlaylistAllowsEmptyAndPreservesDatabaseErrors(t *testing.T) {
	app := newTestApp(t)
	assertEmpty := func() {
		t.Helper()
		response := performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusOK)
		tracks := responseData[[]model.MusicTrack](t, response)
		if tracks == nil || len(tracks) != 0 {
			t.Fatalf("空播放列表应返回非 null 空数组: %s", response.Body.String())
		}
	}

	assertEmpty()
	if err := app.db.Exec("UPDATE playlists SET enabled = ?", false).Error; err != nil {
		t.Fatal(err)
	}
	assertEmpty()
	if err := app.db.Exec("DELETE FROM playlists").Error; err != nil {
		t.Fatal(err)
	}
	assertEmpty()

	sqlDB, err := app.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlist", nil, nil, http.StatusInternalServerError)
}

func TestPublicPlaybackRateLimitIncludesMissingTracks(t *testing.T) {
	app := newTestApp(t)
	for range 120 {
		performJSON(t, app.router, http.MethodGet, "/api/v1/music/tracks/999999/playback", nil, nil, http.StatusNotFound)
	}
	performJSON(t, app.router, http.MethodGet, "/api/v1/music/tracks/999999/playback", nil, nil, http.StatusTooManyRequests)
	performJSON(t, app.router, http.MethodGet, "/api/v1/music/tracks/999999/lyrics", nil, nil, http.StatusNotFound)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/music/tracks/999999/playback", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	response := httptest.NewRecorder()
	app.router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("其他 IP 不应受到限流影响: %d %s", response.Code, response.Body.String())
	}
}

func TestMusicPlaylistsPostSecurityAndLyrics(t *testing.T) {
	app := newTestApp(t)
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/list", nil, nil, http.StatusUnauthorized)
	performJSON(t, app.router, http.MethodGet, "/api/v1/admin/music/tracks/list", nil, nil, http.StatusMethodNotAllowed)
	cookie := login(t, app.router)

	lrc := "[00:00.00]第一行\n[00:02.50]第二行\n"
	performUpload(t, app.router, "/api/v1/admin/uploads/lyrics", "bad.txt", []byte(lrc), cookie, http.StatusBadRequest)
	lyricURL := responseData[struct {
		URL string `json:"url"`
	}](t, performUpload(t, app.router, "/api/v1/admin/uploads/lyrics", "song.lrc", []byte(lrc), cookie, http.StatusOK)).URL

	track := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/create", model.MusicTrack{
		Title: "Local", Artist: "Artist", URL: "/storage/music/test.wav", LyricPath: lyricURL,
		Source: "local", Enabled: true,
	}, cookie, http.StatusOK))
	second := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/create", model.MusicTrack{
		Title: "Second", URL: "/storage/music/second.wav", Source: "local", Enabled: true,
	}, cookie, http.StatusOK))
	playlist := responseData[model.Playlist](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/create", model.Playlist{
		Name: "夜间", Sort: 2, Enabled: true,
	}, cookie, http.StatusOK))
	playlist = responseData[model.Playlist](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/tracks", gin.H{
		"playlist_id": playlist.ID, "track_ids": []uint{second.ID, track.ID, second.ID},
	}, cookie, http.StatusOK))
	if len(playlist.Tracks) != 2 || playlist.Tracks[0].ID != second.ID || playlist.Tracks[1].ID != track.ID {
		t.Fatalf("播放列表关联去重或排序错误: %+v", playlist.Tracks)
	}

	lyrics := responseData[struct {
		Text string `json:"text"`
	}](t, performJSON(t, app.router, http.MethodGet, fmt.Sprintf("/api/v1/music/tracks/%d/lyrics", track.ID), nil, nil, http.StatusOK))
	if lyrics.Text != lrc {
		t.Fatalf("歌词内容错误: %q", lyrics.Text)
	}
	playback := responseData[struct {
		URL string `json:"url"`
	}](t, performJSON(t, app.router, http.MethodGet, fmt.Sprintf("/api/v1/music/tracks/%d/playback", track.ID), nil, nil, http.StatusOK))
	if playback.URL != track.URL {
		t.Fatalf("本地播放地址错误: %q", playback.URL)
	}

	settings := responseData[model.Settings](t, performJSON(t, app.router, http.MethodGet, "/api/v1/settings", nil, nil, http.StatusOK))
	settings.Music.PlaylistID = playlist.ID + 1000
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", settings, cookie, http.StatusNotFound)
	settings.Music.PlaylistID = playlist.ID
	settings.Music.LyricsEnabled = false
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", settings, cookie, http.StatusOK)

	private := responseData[model.MusicTrack](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/tracks/create", model.MusicTrack{
		Title: "Not listed", URL: "/storage/music/private.wav", Source: "local", Enabled: true, PlaylistIDs: []uint{},
	}, cookie, http.StatusOK))
	performJSON(t, app.router, http.MethodGet, fmt.Sprintf("/api/v1/music/tracks/%d/playback", private.ID), nil, nil, http.StatusNotFound)
	legacyTracks := responseData[[]model.MusicTrack](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/tracks", nil, nil, http.StatusOK))
	for _, item := range legacyTracks {
		if item.ID == private.ID {
			t.Fatal("未关联启用播放列表的曲目出现在公开接口")
		}
	}

	playlist.Enabled = false
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/update", playlist, cookie, http.StatusOK)
	performJSON(t, app.router, http.MethodPut, "/api/v1/admin/settings", settings, cookie, http.StatusNotFound)
	performJSON(t, app.router, http.MethodGet, fmt.Sprintf("/api/v1/music/playlists/%d", playlist.ID), nil, nil, http.StatusNotFound)
	public := responseData[[]model.Playlist](t, performJSON(t, app.router, http.MethodGet, "/api/v1/music/playlists", nil, nil, http.StatusOK))
	for _, item := range public {
		if item.ID == playlist.ID {
			t.Fatal("停用播放列表出现在公开接口")
		}
	}
}

func TestPlaylistNameUniqueness(t *testing.T) {
	app := newTestApp(t)
	cookie := login(t, app.router)
	first := responseData[model.Playlist](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/create", model.Playlist{
		Name: "Daily", Enabled: true,
	}, cookie, http.StatusOK))
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/create", model.Playlist{
		Name: " daily ", Enabled: true,
	}, cookie, http.StatusBadRequest)
	second := responseData[model.Playlist](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/create", model.Playlist{
		Name: "Other", Enabled: true,
	}, cookie, http.StatusOK))
	second.Name = " DAILY "
	performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/update", second, cookie, http.StatusBadRequest)
	second.Name = " Other "
	updated := responseData[model.Playlist](t, performJSON(t, app.router, http.MethodPost, "/api/v1/admin/music/playlists/update", second, cookie, http.StatusOK))
	if updated.Name != "Other" || first.Name != "Daily" {
		t.Fatalf("列表名称规范化错误: first=%+v, updated=%+v", first, updated)
	}
}
