package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"homepage/internal/model"
	"homepage/internal/repository"

	"github.com/dhowden/tag"
	"gorm.io/gorm"
)

type Playback struct {
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Lyrics struct {
	Text   string `json:"text"`
	Format string `json:"format"`
}

type Cover struct {
	URL         string
	Data        []byte
	ContentType string
}

const (
	maxCoverBytes      = 10 << 20
	maxCoverAudioBytes = 128 << 20
	missingCoverTTL    = 5 * time.Minute
)

type coverProvider interface {
	Cover(context.Context, model.MusicTrack) (io.ReadCloser, error)
}

type trackCoverProvider interface {
	TrackCover(context.Context, model.MusicTrack) (string, error)
}

type MusicProvider interface {
	Playback(context.Context, model.MusicTrack) (Playback, error)
	Lyrics(context.Context, model.MusicTrack) (string, error)
}

type MediaPolicy struct {
	AllowInsecureHTTP bool
}

func (p MediaPolicy) AllowBrowserURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "https" || (parsed.Scheme == "http" && p.AllowInsecureHTTP)
}

func (p MediaPolicy) ValidatePlaybackURL(value string) error {
	if !p.AllowBrowserURL(value) {
		return failure(http.StatusUnprocessableEntity, "playback_unavailable", "媒体地址必须使用 HTTPS；受控 HTTP 部署可显式允许不安全媒体")
	}
	return nil
}

type MusicService struct {
	repo          *repository.Repository
	root          string
	providers     map[string]MusicProvider
	mediaPolicy   MediaPolicy
	limit         *IPRateLimiter
	playbackLimit *IPRateLimiter
	lyricsLimit   *IPRateLimiter
	coverLimit    *IPRateLimiter
	coverMu       sync.Mutex
	missingCover  map[uint]time.Time
}

func NewMusicService(repo *repository.Repository, storageRoot string) *MusicService {
	return &MusicService{
		repo: repo, root: storageRoot, providers: make(map[string]MusicProvider),
		limit: NewIPRateLimiter(120, time.Minute), playbackLimit: NewIPRateLimiter(120, time.Minute),
		lyricsLimit: NewIPRateLimiter(120, time.Minute), coverLimit: NewIPRateLimiter(300, time.Minute),
		missingCover: make(map[uint]time.Time),
	}
}

func (s *MusicService) RegisterProvider(source string, provider MusicProvider) {
	s.providers[source] = provider
}

func (s *MusicService) SetMediaPolicy(policy MediaPolicy) {
	s.mediaPolicy = policy
}

func (s *MusicService) Allow(ip string) bool { return s.limit.Allow(ip) }

func (s *MusicService) AllowPlayback(ip string) bool { return s.playbackLimit.Allow(ip) }

func (s *MusicService) AllowLyrics(ip string) bool { return s.lyricsLimit.Allow(ip) }

func (s *MusicService) AllowCover(ip string) bool { return s.coverLimit.Allow(ip) }

func (s *MusicService) Playback(ctx context.Context, id uint) (Playback, error) {
	track, err := s.publicTrack(id)
	if err != nil {
		return Playback{}, err
	}
	switch track.Source {
	case "local":
		return Playback{URL: track.URL}, nil
	case "remote":
		if err := s.mediaPolicy.ValidatePlaybackURL(track.URL); err != nil {
			return Playback{}, err
		}
		return Playback{URL: track.URL}, nil
	default:
		provider, ok := s.providers[track.Source]
		if !ok {
			return Playback{}, failure(http.StatusServiceUnavailable, "provider_unavailable", "音乐来源尚未配置")
		}
		playback, err := provider.Playback(ctx, track)
		if err != nil {
			return Playback{}, err
		}
		if err := s.mediaPolicy.ValidatePlaybackURL(playback.URL); err != nil {
			return Playback{}, err
		}
		return playback, nil
	}
}

func (s *MusicService) Lyrics(ctx context.Context, id uint) (Lyrics, error) {
	track, err := s.publicTrack(id)
	if err != nil {
		return Lyrics{}, err
	}
	result := Lyrics{Format: "lrc"}
	switch track.Source {
	case "local":
		if track.LyricPath == "" {
			return result, nil
		}
		text, err := s.localLyrics(track.LyricPath)
		result.Text = text
		return result, err
	case "remote":
		return result, nil
	default:
		provider, ok := s.providers[track.Source]
		if !ok {
			return Lyrics{}, failure(http.StatusServiceUnavailable, "provider_unavailable", "音乐来源尚未配置")
		}
		result.Text, err = provider.Lyrics(ctx, track)
		return result, err
	}
}

func (s *MusicService) neteaseCover(ctx context.Context, track model.MusicTrack) (Cover, error) {
	s.coverMu.Lock()
	expiresAt, missing := s.missingCover[track.ID]
	if missing && time.Now().Before(expiresAt) {
		s.coverMu.Unlock()
		return Cover{}, notFound("封面不存在")
	}
	delete(s.missingCover, track.ID)
	s.coverMu.Unlock()

	provider, ok := s.providers[neteaseProvider].(trackCoverProvider)
	if !ok {
		return Cover{}, notFound("封面不存在")
	}
	cover, err := provider.TrackCover(ctx, track)
	if err != nil {
		var serviceError *Error
		if errors.As(err, &serviceError) && serviceError.Status == http.StatusNotFound {
			s.markCoverMissing(track.ID)
		}
		return Cover{}, err
	}
	if !s.mediaPolicy.AllowBrowserURL(cover) {
		s.markCoverMissing(track.ID)
		return Cover{}, notFound("封面不存在或地址不受当前媒体安全策略支持")
	}
	if err := s.repo.FillTrackCoverIfEmpty(track.ID, cover); err != nil {
		return Cover{}, err
	}
	return Cover{URL: cover}, nil
}

func (s *MusicService) markCoverMissing(id uint) {
	s.coverMu.Lock()
	s.missingCover[id] = time.Now().Add(missingCoverTTL)
	s.coverMu.Unlock()
}

func (s *MusicService) Cover(ctx context.Context, id uint) (Cover, error) {
	track, err := s.publicTrack(id)
	if err != nil {
		return Cover{}, err
	}
	if httpURL(track.Cover) && oneOf(track.Source, "netease", "remote") {
		if !s.mediaPolicy.AllowBrowserURL(track.Cover) {
			return Cover{}, notFound("封面地址不受当前媒体安全策略支持")
		}
		return Cover{URL: track.Cover}, nil
	}
	if track.Source == "netease" {
		return s.neteaseCover(ctx, track)
	}
	if track.Source == "remote" {
		return Cover{}, notFound("封面不存在")
	}

	cachePath := filepath.Join(s.root, "covers", fmt.Sprintf("%d-%d.img", track.ID, track.UpdatedAt.UnixNano()))
	if data, readErr := os.ReadFile(cachePath); readErr == nil {
		if contentType, ok := coverContentType(data); ok {
			return Cover{Data: data, ContentType: contentType}, nil
		}
	}

	var cover Cover
	switch track.Source {
	case "local":
		cover, err = s.localCover(track.URL)
	default:
		provider, ok := s.providers[track.Source].(coverProvider)
		if !ok {
			return Cover{}, notFound("封面不存在")
		}
		cover, err = providerCover(ctx, provider, track)
	}
	if err != nil {
		return Cover{}, err
	}
	if err := writeCoverCache(cachePath, cover.Data); err != nil {
		return Cover{}, err
	}
	return cover, nil
}

func (s *MusicService) localCover(resource string) (Cover, error) {
	name := filepath.Base(resource)
	if name == "." || name == "" || resource != "/storage/music/"+name {
		return Cover{}, invalid("音乐路径无效")
	}
	file, err := os.Open(filepath.Join(s.root, "music", name))
	if errors.Is(err, os.ErrNotExist) {
		return Cover{}, notFound("音乐文件不存在")
	}
	if err != nil {
		return Cover{}, err
	}
	defer file.Close()
	return embeddedCover(file)
}

func providerCover(ctx context.Context, provider coverProvider, track model.MusicTrack) (Cover, error) {
	reader, err := provider.Cover(ctx, track)
	if err != nil {
		return Cover{}, err
	}
	defer reader.Close()
	file, err := os.CreateTemp("", "homepage-cover-*.audio")
	if err != nil {
		return Cover{}, err
	}
	name := file.Name()
	defer os.Remove(name)
	defer file.Close()
	written, err := io.Copy(file, io.LimitReader(reader, maxCoverAudioBytes+1))
	if err != nil {
		return Cover{}, err
	}
	if written > maxCoverAudioBytes {
		return Cover{}, invalid("封面音频超过大小限制")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Cover{}, err
	}
	return embeddedCover(file)
}

func embeddedCover(reader io.ReadSeeker) (Cover, error) {
	metadata, err := tag.ReadFrom(reader)
	if err != nil || metadata.Picture() == nil {
		return Cover{}, notFound("封面不存在")
	}
	data := metadata.Picture().Data
	contentType, ok := coverContentType(data)
	if !ok {
		return Cover{}, notFound("封面不存在或格式不受支持")
	}
	return Cover{Data: data, ContentType: contentType}, nil
}

func coverContentType(data []byte) (string, bool) {
	if len(data) == 0 || len(data) > maxCoverBytes {
		return "", false
	}
	contentType := http.DetectContentType(data)
	return contentType, oneOf(contentType, "image/jpeg", "image/png", "image/gif", "image/webp")
}

func writeCoverCache(name string, data []byte) error {
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".cover-*")
	if err != nil {
		return err
	}
	tempName := file.Name()
	defer os.Remove(tempName)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, name); err != nil {
		if _, statErr := os.Stat(name); statErr == nil {
			return nil
		}
		return err
	}
	return nil
}

func (s *MusicService) publicTrack(id uint) (model.MusicTrack, error) {
	track, err := s.repo.PublicTrackByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return track, notFound("曲目不存在或未公开")
	}
	return track, err
}

func (s *MusicService) localLyrics(resource string) (string, error) {
	filename, err := regularLocalResourcePath(s.root, "lyrics", resource)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 1<<20 {
		return "", invalid("歌词文件超过大小限制")
	}
	return string(data), nil
}
