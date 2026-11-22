package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
)

type coverProvider interface {
	Cover(context.Context, model.MusicTrack) (io.ReadCloser, error)
}

type MusicProvider interface {
	Playback(context.Context, model.MusicTrack) (Playback, error)
	Lyrics(context.Context, model.MusicTrack) (string, error)
}

type MusicService struct {
	repo      *repository.Repository
	root      string
	providers map[string]MusicProvider
	limit     *IPRateLimiter
}

func NewMusicService(repo *repository.Repository, storageRoot string) *MusicService {
	return &MusicService{
		repo: repo, root: storageRoot, providers: make(map[string]MusicProvider),
		limit: NewIPRateLimiter(120, time.Minute),
	}
}

func (s *MusicService) RegisterProvider(source string, provider MusicProvider) {
	s.providers[source] = provider
}

func (s *MusicService) Allow(ip string) bool { return s.limit.Allow(ip) }

func (s *MusicService) Playback(ctx context.Context, id uint) (Playback, error) {
	track, err := s.publicTrack(id)
	if err != nil {
		return Playback{}, err
	}
	switch track.Source {
	case "local", "remote":
		return Playback{URL: track.URL}, nil
	default:
		provider, ok := s.providers[track.Source]
		if !ok {
			return Playback{}, failure(http.StatusServiceUnavailable, "provider_unavailable", "音乐来源尚未配置")
		}
		return provider.Playback(ctx, track)
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

func (s *MusicService) Cover(ctx context.Context, id uint) (Cover, error) {
	track, err := s.publicTrack(id)
	if err != nil {
		return Cover{}, err
	}
	if oneOf(track.Source, "netease", "remote") {
		if httpURL(track.Cover) {
			return Cover{URL: track.Cover}, nil
		}
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
	name := filepath.Base(resource)
	if name == "." || name == "" || resource != "/storage/lyrics/"+name {
		return "", invalid("歌词路径无效")
	}
	file, err := os.Open(filepath.Join(s.root, "lyrics", name))
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
