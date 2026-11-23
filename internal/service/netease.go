package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"homepage/internal/model"
	"homepage/internal/repository"

	"gorm.io/gorm"
)

const (
	neteaseProvider    = "netease"
	neteaseHTTPTimeout = 15 * time.Second
	neteaseMaxResponse = 8 << 20
	neteaseQRTTL       = 2 * time.Minute
	neteaseLyricsTTL   = 5 * time.Minute
)

type NeteaseQRCreateResult struct {
	SessionID    string    `json:"session_id"`
	QRCodeBase64 string    `json:"qr_code_base64"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type NeteaseQRCheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type NeteaseProfile struct {
	UserID    int64  `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type NeteaseStatus struct {
	Authenticated bool            `json:"authenticated"`
	Profile       *NeteaseProfile `json:"profile,omitempty"`
}

type NeteaseSearchResult struct {
	Tracks   []model.MusicTrack `json:"tracks"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
}

type NeteaseConfig struct {
	BaseURL string `json:"base_url"`
}

type neteaseProviderConfig struct {
	BaseURL string `json:"base_url"`
	Cookie  string `json:"cookie"`
}

type neteaseQRSession struct {
	key       string
	expiresAt time.Time
}

type neteaseLyricsEntry struct {
	text      string
	expiresAt time.Time
}

type NeteaseService struct {
	repo     *repository.Repository
	client   *http.Client
	mu       sync.Mutex
	sessions map[string]neteaseQRSession
	lyricsMu sync.Mutex
	lyrics   map[string]neteaseLyricsEntry
}

func NewNeteaseService(repo *repository.Repository, client *http.Client) *NeteaseService {
	if client == nil {
		client = &http.Client{Timeout: neteaseHTTPTimeout}
	} else {
		clone := *client
		if clone.Timeout <= 0 {
			clone.Timeout = neteaseHTTPTimeout
		}
		client = &clone
	}
	return &NeteaseService{
		repo: repo, client: client,
		sessions: make(map[string]neteaseQRSession), lyrics: make(map[string]neteaseLyricsEntry),
	}
}

func (s *NeteaseService) ReadConfig() (NeteaseConfig, error) {
	config, err := s.loadProviderConfig()
	if err != nil {
		return NeteaseConfig{}, err
	}
	return NeteaseConfig{BaseURL: config.BaseURL}, nil
}

func (s *NeteaseService) SaveConfig(input NeteaseConfig) error {
	parsed, err := parseNeteaseBaseURL(input.BaseURL)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.providerConfig()
	if err != nil {
		return err
	}
	baseURL := parsed.String()
	if current.BaseURL != baseURL {
		current.Cookie = ""
		s.sessions = make(map[string]neteaseQRSession)
		s.lyricsMu.Lock()
		s.lyrics = make(map[string]neteaseLyricsEntry)
		s.lyricsMu.Unlock()
	}
	current.BaseURL = baseURL
	return s.repo.SaveProviderConfig(neteaseProvider, current)
}

func (s *NeteaseService) QRCreate(ctx context.Context) (NeteaseQRCreateResult, error) {
	var keyResponse struct {
		Code int `json:"code"`
		Data struct {
			Unikey string `json:"unikey"`
		} `json:"data"`
	}
	query := url.Values{"timestamp": {strconv.FormatInt(time.Now().UnixMilli(), 10)}}
	if err := s.request(ctx, http.MethodGet, "/login/qr/key", query, &keyResponse); err != nil {
		return NeteaseQRCreateResult{}, err
	}
	if keyResponse.Code != http.StatusOK || keyResponse.Data.Unikey == "" {
		return NeteaseQRCreateResult{}, neteaseUpstreamError()
	}

	var qrResponse struct {
		Code int `json:"code"`
		Data struct {
			QRImage string `json:"qrimg"`
		} `json:"data"`
	}
	query = url.Values{
		"key":       {keyResponse.Data.Unikey},
		"qrimg":     {"true"},
		"timestamp": {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}
	if err := s.request(ctx, http.MethodGet, "/login/qr/create", query, &qrResponse); err != nil {
		return NeteaseQRCreateResult{}, err
	}
	if qrResponse.Code != http.StatusOK || qrResponse.Data.QRImage == "" {
		return NeteaseQRCreateResult{}, neteaseUpstreamError()
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return NeteaseQRCreateResult{}, err
	}
	result := NeteaseQRCreateResult{
		SessionID: base64.RawURLEncoding.EncodeToString(bytes), QRCodeBase64: qrResponse.Data.QRImage,
		ExpiresAt: time.Now().Add(neteaseQRTTL),
	}
	now := time.Now()
	s.mu.Lock()
	for id, session := range s.sessions {
		if !now.Before(session.expiresAt) {
			delete(s.sessions, id)
		}
	}
	s.sessions[result.SessionID] = neteaseQRSession{key: keyResponse.Data.Unikey, expiresAt: result.ExpiresAt}
	s.mu.Unlock()
	return result, nil
}

func (s *NeteaseService) QRCheck(ctx context.Context, sessionID string) (NeteaseQRCheckResult, error) {
	now := time.Now()
	s.mu.Lock()
	for id, session := range s.sessions {
		if !now.Before(session.expiresAt) {
			delete(s.sessions, id)
		}
	}
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return NeteaseQRCheckResult{}, invalid("二维码会话无效或已过期")
	}

	var response struct {
		Code   int    `json:"code"`
		Cookie string `json:"cookie"`
	}
	query := url.Values{
		"key": {session.key}, "timestamp": {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}
	if err := s.request(ctx, http.MethodGet, "/login/qr/check", query, &response); err != nil {
		return NeteaseQRCheckResult{}, err
	}
	switch response.Code {
	case 800:
		s.deleteQRSession(sessionID)
		return NeteaseQRCheckResult{Status: "expired", Message: "二维码已过期，请重新生成"}, nil
	case 801:
		return NeteaseQRCheckResult{Status: "waiting", Message: "等待扫码"}, nil
	case 802:
		return NeteaseQRCheckResult{Status: "scanned", Message: "已扫码，等待确认"}, nil
	case 803:
		if response.Cookie == "" {
			return NeteaseQRCheckResult{}, failure(http.StatusBadGateway, "upstream_error", "网易云登录响应缺少 Cookie")
		}
		if err := s.saveCookie(response.Cookie); err != nil {
			return NeteaseQRCheckResult{}, err
		}
		s.deleteQRSession(sessionID)
		return NeteaseQRCheckResult{Status: "authenticated", Message: "登录成功"}, nil
	default:
		return NeteaseQRCheckResult{}, neteaseUpstreamError()
	}
}

func (s *NeteaseService) Status(ctx context.Context) (NeteaseStatus, error) {
	var response struct {
		Code    int             `json:"code"`
		Profile *neteaseProfile `json:"profile"`
		Data    struct {
			Code    int             `json:"code"`
			Profile *neteaseProfile `json:"profile"`
		} `json:"data"`
	}
	query := url.Values{"timestamp": {strconv.FormatInt(time.Now().UnixMilli(), 10)}}
	if err := s.request(ctx, http.MethodPost, "/login/status", query, &response); err != nil {
		return NeteaseStatus{}, err
	}
	profile := response.Data.Profile
	if profile == nil {
		profile = response.Profile
	}
	if profile == nil {
		return NeteaseStatus{Authenticated: false}, nil
	}
	return NeteaseStatus{Authenticated: true, Profile: &NeteaseProfile{
		UserID: profile.UserID, Nickname: profile.Nickname, AvatarURL: profile.AvatarURL,
	}}, nil
}

func (s *NeteaseService) Logout(ctx context.Context) error {
	query := url.Values{"timestamp": {strconv.FormatInt(time.Now().UnixMilli(), 10)}}
	upstreamErr := s.request(ctx, http.MethodPost, "/logout", query, nil)
	if err := s.saveCookie(""); err != nil {
		return err
	}
	return upstreamErr
}

func (s *NeteaseService) Search(ctx context.Context, keywords string, page, pageSize int) (NeteaseSearchResult, error) {
	keywords = strings.TrimSpace(keywords)
	if !required(keywords, 200) {
		return NeteaseSearchResult{}, invalid("搜索关键词无效")
	}
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		return NeteaseSearchResult{}, invalid("搜索分页超过限制")
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	if offset < 0 {
		return NeteaseSearchResult{}, invalid("搜索分页无效")
	}
	var response struct {
		Code   int `json:"code"`
		Result struct {
			Songs     []neteaseSong `json:"songs"`
			SongCount int           `json:"songCount"`
		} `json:"result"`
	}
	query := url.Values{
		"keywords": {keywords}, "limit": {strconv.Itoa(pageSize)}, "offset": {strconv.Itoa(offset)},
	}
	if err := s.request(ctx, http.MethodGet, "/search", query, &response); err != nil {
		return NeteaseSearchResult{}, err
	}
	if response.Code != http.StatusOK {
		return NeteaseSearchResult{}, neteaseUpstreamError()
	}
	tracks := make([]model.MusicTrack, len(response.Result.Songs))
	for i, song := range response.Result.Songs {
		tracks[i] = neteaseTrack(song, offset+i)
	}
	return NeteaseSearchResult{
		Tracks: tracks, Page: page, PageSize: pageSize, Total: response.Result.SongCount,
	}, nil
}

func (s *NeteaseService) CreateFromTracks(name string, sort int, tracks []model.MusicTrack) (model.Playlist, int, error) {
	playlist := model.Playlist{Name: name, Sort: sort, Enabled: true}
	if err := validatePlaylist(&playlist); err != nil {
		return model.Playlist{}, 0, err
	}
	items, err := normalizeNeteaseTracks(tracks)
	if err != nil {
		return model.Playlist{}, 0, err
	}
	playlist, duplicates, err := s.repo.ImportPlaylist(playlist, items)
	return playlist, duplicates, playlistStoreError(err)
}

func (s *NeteaseService) AppendToPlaylist(playlistID uint, tracks []model.MusicTrack) (model.Playlist, int, int, error) {
	items, err := normalizeNeteaseTracks(tracks)
	if err != nil {
		return model.Playlist{}, 0, 0, err
	}
	playlist, imported, duplicates, err := s.repo.AppendPlaylistTracks(playlistID, items)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Playlist{}, 0, 0, notFound("播放列表不存在")
	}
	return playlist, imported, duplicates, err
}

func normalizeNeteaseTracks(tracks []model.MusicTrack) ([]model.MusicTrack, error) {
	if len(tracks) == 0 || len(tracks) > 1000 {
		return nil, invalid("请选择 1 至 1000 首网易云歌曲")
	}
	items := append([]model.MusicTrack(nil), tracks...)
	for i := range items {
		items[i].ID = 0
		items[i].PlaylistIDs = nil
		items[i].Sort = i
		if items[i].Source != neteaseProvider {
			return nil, invalid("只能导入网易云歌曲")
		}
		externalID, ok := parsePositiveNeteaseID(items[i].ExternalID)
		if !ok {
			return nil, invalid("网易云歌曲 ID 无效")
		}
		items[i].ExternalID = externalID
		if err := validateTrack(items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *NeteaseService) ImportPlaylist(ctx context.Context, input, name string) (model.Playlist, int, error) {
	id, ok := parseNeteasePlaylistID(input)
	if !ok {
		return model.Playlist{}, 0, invalid("网易云歌单 ID 或分享链接无效")
	}
	var detail struct {
		Code     int `json:"code"`
		Playlist *struct {
			Name     string `json:"name"`
			TrackIDs []struct {
				ID int64 `json:"id"`
			} `json:"trackIds"`
		} `json:"playlist"`
	}
	if err := s.request(ctx, http.MethodGet, "/playlist/detail", url.Values{"id": {id}}, &detail); err != nil {
		return model.Playlist{}, 0, err
	}
	if detail.Code == http.StatusNotFound || detail.Playlist == nil {
		return model.Playlist{}, 0, notFound("网易云歌单不存在或不可访问")
	}
	if detail.Code != http.StatusOK {
		return model.Playlist{}, 0, neteaseUpstreamError()
	}
	if strings.TrimSpace(name) == "" {
		name = detail.Playlist.Name
	}

	const limit = 100
	total := len(detail.Playlist.TrackIDs)
	tracks := make([]model.MusicTrack, 0, total)
	for offset := 0; ; {
		var response struct {
			Code  int           `json:"code"`
			Songs []neteaseSong `json:"songs"`
		}
		query := url.Values{
			"id": {id}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)},
		}
		if err := s.request(ctx, http.MethodGet, "/playlist/track/all", query, &response); err != nil {
			return model.Playlist{}, 0, err
		}
		if response.Code != http.StatusOK {
			return model.Playlist{}, 0, neteaseUpstreamError()
		}
		for _, song := range response.Songs {
			tracks = append(tracks, neteaseTrack(song, len(tracks)))
		}
		if len(response.Songs) == 0 {
			break
		}
		offset += len(response.Songs)
		if total > 0 && offset >= total {
			break
		}
		if total == 0 && len(response.Songs) < limit {
			break
		}
	}
	if total > len(tracks) {
		return model.Playlist{}, 0, failure(http.StatusBadGateway, "upstream_error", "网易云歌单曲目响应不完整")
	}
	return s.CreateFromTracks(name, 0, tracks)
}

func (s *NeteaseService) TrackCover(ctx context.Context, track model.MusicTrack) (string, error) {
	id, ok := parsePositiveNeteaseID(track.ExternalID)
	if !ok || track.Source != neteaseProvider {
		return "", invalid("网易云歌曲 ID 无效")
	}
	var response struct {
		Code  int           `json:"code"`
		Songs []neteaseSong `json:"songs"`
	}
	if err := s.request(ctx, http.MethodGet, "/song/detail", url.Values{"ids": {id}}, &response); err != nil {
		return "", err
	}
	if response.Code != http.StatusOK {
		return "", neteaseUpstreamError()
	}
	if len(response.Songs) == 0 {
		return "", notFound("封面不存在")
	}
	cover := neteaseTrack(response.Songs[0], 0).Cover
	if !httpURL(cover) {
		return "", notFound("封面不存在")
	}
	return cover, nil
}

func (s *NeteaseService) Playback(ctx context.Context, track model.MusicTrack) (Playback, error) {
	id, ok := parsePositiveNeteaseID(track.ExternalID)
	if !ok || track.Source != neteaseProvider {
		return Playback{}, invalid("网易云歌曲 ID 无效")
	}
	var response struct {
		Code int `json:"code"`
		Data []struct {
			URL  string `json:"url"`
			Expi int    `json:"expi"`
		} `json:"data"`
	}
	query := url.Values{"id": {id}, "level": {"standard"}}
	if err := s.request(ctx, http.MethodGet, "/song/url/v1", query, &response); err != nil {
		return Playback{}, err
	}
	if response.Code != http.StatusOK || len(response.Data) == 0 || response.Data[0].URL == "" {
		return Playback{}, failure(http.StatusUnprocessableEntity, "playback_unavailable", "该歌曲暂无可播放地址，可能受版权或会员权限限制")
	}
	playback := Playback{URL: normalizeNeteaseMediaURL(response.Data[0].URL)}
	if response.Data[0].Expi > 0 {
		expiresAt := time.Now().Add(time.Duration(response.Data[0].Expi) * time.Second)
		playback.ExpiresAt = &expiresAt
	}
	return playback, nil
}

func (s *NeteaseService) Lyrics(ctx context.Context, track model.MusicTrack) (string, error) {
	id, ok := parsePositiveNeteaseID(track.ExternalID)
	if !ok || track.Source != neteaseProvider {
		return "", invalid("网易云歌曲 ID 无效")
	}
	now := time.Now()
	s.lyricsMu.Lock()
	for key, entry := range s.lyrics {
		if !now.Before(entry.expiresAt) {
			delete(s.lyrics, key)
		}
	}
	if entry, exists := s.lyrics[id]; exists {
		s.lyricsMu.Unlock()
		return entry.text, nil
	}
	s.lyricsMu.Unlock()

	var response struct {
		Code int `json:"code"`
		LRC  struct {
			Lyric string `json:"lyric"`
		} `json:"lrc"`
	}
	if err := s.request(ctx, http.MethodGet, "/lyric", url.Values{"id": {id}}, &response); err != nil {
		return "", err
	}
	if response.Code != http.StatusOK {
		return "", neteaseUpstreamError()
	}
	s.lyricsMu.Lock()
	s.lyrics[id] = neteaseLyricsEntry{text: response.LRC.Lyric, expiresAt: time.Now().Add(neteaseLyricsTTL)}
	s.lyricsMu.Unlock()
	return response.LRC.Lyric, nil
}

type neteaseProfile struct {
	UserID    int64  `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

type neteaseArtist struct {
	Name string `json:"name"`
}

type neteaseAlbum struct {
	Name   string `json:"name"`
	PicURL string `json:"picUrl"`
}

type neteaseSong struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Artists       []neteaseArtist `json:"ar"`
	LegacyArtists []neteaseArtist `json:"artists"`
	Album         neteaseAlbum    `json:"al"`
	LegacyAlbum   neteaseAlbum    `json:"album"`
}

func normalizeNeteaseMediaURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return raw
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "126.net" && !strings.HasSuffix(host, ".126.net") {
		return raw
	}
	parsed.Scheme = "https"
	return parsed.String()
}

func neteaseTrack(song neteaseSong, sort int) model.MusicTrack {
	artists := song.Artists
	if len(artists) == 0 {
		artists = song.LegacyArtists
	}
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	album := song.Album
	if album.Name == "" {
		album.Name = song.LegacyAlbum.Name
	}
	if album.PicURL == "" {
		album.PicURL = song.LegacyAlbum.PicURL
	}
	return model.MusicTrack{
		Title: song.Name, Artist: strings.Join(names, " / "), Album: album.Name, Cover: normalizeNeteaseMediaURL(album.PicURL),
		Source: neteaseProvider, ExternalID: strconv.FormatInt(song.ID, 10), Sort: sort, Enabled: true,
	}
}

func parseNeteasePlaylistID(input string) (string, bool) {
	input = strings.TrimSpace(input)
	if id, ok := parsePositiveNeteaseID(input); ok {
		return id, true
	}
	start := strings.Index(input, "https://")
	if httpStart := strings.Index(input, "http://"); start < 0 || (httpStart >= 0 && httpStart < start) {
		start = httpStart
	}
	if start < 0 {
		return "", false
	}
	candidate := strings.Fields(input[start:])[0]
	candidate = strings.TrimRight(candidate, "<>[]{}()（）【】\"'，。！；;")
	parsed, err := url.Parse(candidate)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	if id, ok := parsePositiveNeteaseID(parsed.Query().Get("id")); ok {
		return id, true
	}
	if index := strings.Index(parsed.Fragment, "?"); index >= 0 {
		values, err := url.ParseQuery(parsed.Fragment[index+1:])
		if err == nil {
			if id, ok := parsePositiveNeteaseID(values.Get("id")); ok {
				return id, true
			}
		}
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "playlist" {
			return parsePositiveNeteaseID(parts[i+1])
		}
	}
	return "", false
}

func parsePositiveNeteaseID(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return "", false
		}
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return "", false
	}
	return strconv.FormatInt(id, 10), true
}

func (s *NeteaseService) request(ctx context.Context, method, endpoint string, query url.Values, target any) error {
	config, err := s.loadProviderConfig()
	if err != nil {
		return err
	}
	baseURL, err := parseNeteaseBaseURL(config.BaseURL)
	if err != nil {
		return failure(http.StatusServiceUnavailable, "provider_not_configured", "请先在后台配置网易云 API 地址")
	}
	requestURL := *baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + endpoint
	requestURL.RawPath = ""
	requestURL.RawQuery = query.Encode()
	requestURL.Fragment = ""
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return err
	}
	if config.Cookie != "" {
		req.Header.Set("Cookie", config.Cookie)
	}
	response, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return failure(http.StatusBadGateway, "upstream_error", "网易云上游请求失败")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, neteaseMaxResponse+1))
	if err != nil {
		return failure(http.StatusBadGateway, "upstream_error", "网易云上游响应读取失败")
	}
	if len(body) > neteaseMaxResponse {
		return failure(http.StatusBadGateway, "upstream_error", "网易云上游响应超过 8MB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return neteaseUpstreamError()
	}
	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return failure(http.StatusBadGateway, "upstream_error", "网易云上游响应无效")
		}
	}
	return nil
}

func (s *NeteaseService) loadProviderConfig() (neteaseProviderConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.providerConfig()
}

func (s *NeteaseService) providerConfig() (neteaseProviderConfig, error) {
	if s.repo == nil {
		return neteaseProviderConfig{}, failure(http.StatusInternalServerError, "provider_unavailable", "网易云配置存储尚未初始化")
	}
	var config neteaseProviderConfig
	if err := s.repo.ProviderConfig(neteaseProvider, &config); err != nil {
		return neteaseProviderConfig{}, err
	}
	return config, nil
}

func parseNeteaseBaseURL(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, invalid("请填写网易云 API 地址")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, invalid("网易云 API 地址无效")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed, nil
}

func (s *NeteaseService) saveCookie(cookie string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.providerConfig()
	if err != nil {
		return err
	}
	if config.BaseURL == "" {
		return failure(http.StatusServiceUnavailable, "provider_not_configured", "请先在后台配置网易云 API 地址")
	}
	config.Cookie = cookie
	return s.repo.SaveProviderConfig(neteaseProvider, config)
}

func (s *NeteaseService) deleteQRSession(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

func neteaseUpstreamError() *Error {
	return failure(http.StatusBadGateway, "upstream_error", "网易云上游返回异常")
}

var _ MusicProvider = (*NeteaseService)(nil)
