package handler

import (
	"fmt"
	"net/http"
	"testing"

	"homepage/internal/model"

	"github.com/gin-gonic/gin"
)

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
