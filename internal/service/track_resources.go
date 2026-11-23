package service

import (
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"homepage/internal/model"
	"homepage/internal/repository"
)

type TrackResourceAuditItem struct {
	Path     string `json:"path"`
	TrackIDs []uint `json:"track_ids"`
}

type TrackResourceAuditReport struct {
	Scanned    int                      `json:"scanned"`
	Referenced int                      `json:"referenced"`
	Orphans    []TrackResourceAuditItem `json:"orphans"`
	Shared     []TrackResourceAuditItem `json:"shared"`
	Invalid    []TrackResourceAuditItem `json:"invalid_names"`
	NonRegular []TrackResourceAuditItem `json:"non_regular"`
}

func (s *ContentService) AuditTrackResources() (TrackResourceAuditReport, error) {
	references, err := s.repo.TrackResourceReferences()
	if err != nil {
		return TrackResourceAuditReport{}, err
	}
	return s.uploads.AuditTrackResources(references)
}

func (s *UploadService) AuditTrackResources(references []repository.TrackResourceReference) (TrackResourceAuditReport, error) {
	report := TrackResourceAuditReport{
		Orphans: make([]TrackResourceAuditItem, 0), Shared: make([]TrackResourceAuditItem, 0),
		Invalid: make([]TrackResourceAuditItem, 0), NonRegular: make([]TrackResourceAuditItem, 0),
	}
	used := make(map[string]map[uint]struct{})
	trackIDs := make(map[uint]struct{}, len(references))
	for _, reference := range references {
		trackIDs[reference.ID] = struct{}{}
		for _, resource := range []string{reference.URL, reference.LyricPath} {
			if resource == "" {
				continue
			}
			if used[resource] == nil {
				used[resource] = make(map[uint]struct{})
			}
			used[resource][reference.ID] = struct{}{}
		}
	}
	for _, kind := range []string{"music", "lyrics", "covers"} {
		entries, err := os.ReadDir(filepath.Join(s.root, kind))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return report, err
		}
		for _, entry := range entries {
			report.Scanned++
			resource := "/storage/" + kind + "/" + entry.Name()
			item := TrackResourceAuditItem{Path: resource, TrackIDs: make([]uint, 0)}
			if !validTrackResourceName(kind, entry.Name()) {
				report.Invalid = append(report.Invalid, item)
				continue
			}
			info, err := os.Lstat(filepath.Join(s.root, kind, entry.Name()))
			if err != nil {
				return report, err
			}
			if !info.Mode().IsRegular() {
				report.NonRegular = append(report.NonRegular, item)
				continue
			}
			if kind == "covers" {
				trackID, _ := coverCacheTrackID(entry.Name())
				if _, ok := trackIDs[trackID]; ok {
					item.TrackIDs = append(item.TrackIDs, trackID)
				}
			} else {
				for trackID := range used[resource] {
					item.TrackIDs = append(item.TrackIDs, trackID)
				}
				sort.Slice(item.TrackIDs, func(i, j int) bool { return item.TrackIDs[i] < item.TrackIDs[j] })
			}
			switch len(item.TrackIDs) {
			case 0:
				report.Orphans = append(report.Orphans, item)
			case 1:
				report.Referenced++
			default:
				report.Shared = append(report.Shared, item)
			}
		}
	}
	return report, nil
}

func validTrackResourceName(kind, name string) bool {
	if kind == "covers" {
		_, ok := coverCacheTrackID(name)
		return ok
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	ext := path.Ext(name)
	if kind == "lyrics" {
		if ext != ".lrc" {
			return false
		}
	} else if kind != "music" || !allowedExtension("music", ext) || ext != strings.ToLower(ext) {
		return false
	}
	encoded := strings.TrimSuffix(name, ext)
	if len(encoded) != 32 || len(name) != 32+len(ext) || encoded != strings.ToLower(encoded) {
		return false
	}
	_, err := hex.DecodeString(encoded)
	return err == nil
}

func validLyricResourcePath(resource string) bool {
	if resource == "" {
		return true
	}
	_, err := localResourceName("lyrics", resource)
	return err == nil
}

func (s *ContentService) validateLyricFile(resource string) error {
	if resource == "" || s.uploads == nil {
		return nil
	}
	_, err := regularLocalResourcePath(s.uploads.root, "lyrics", resource)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func localResourceName(kind, resource string) (string, error) {
	prefix := "/storage/" + kind + "/"
	if !strings.HasPrefix(resource, prefix) {
		return "", invalid("本地资源路径无效")
	}
	name := strings.TrimPrefix(resource, prefix)
	if !validTrackResourceName(kind, name) {
		return "", invalid("本地资源路径无效")
	}
	return name, nil
}

func regularLocalResourcePath(root, kind, resource string) (string, error) {
	name, err := localResourceName(kind, resource)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(root, kind, name)
	info, err := os.Lstat(filename)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", invalid("本地资源不是普通文件")
	}
	return filename, nil
}

func (s *UploadService) deleteLocalResource(kind, resource string) error {
	filename, err := regularLocalResourcePath(s.root, kind, resource)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *UploadService) DeleteLyrics(resource string) error {
	return s.deleteLocalResource("lyrics", resource)
}

func (s *UploadService) DeleteTrackCovers(trackID uint) error {
	entries, err := os.ReadDir(filepath.Join(s.root, "covers"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var result error
	prefix := strconv.FormatUint(uint64(trackID), 10) + "-"
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) || !validTrackResourceName("covers", entry.Name()) {
			continue
		}
		id, _ := coverCacheTrackID(entry.Name())
		if id != trackID {
			continue
		}
		filename := filepath.Join(s.root, "covers", entry.Name())
		info, err := os.Lstat(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			result = errors.Join(result, err)
			continue
		}
		if !info.Mode().IsRegular() {
			result = errors.Join(result, invalid("封面缓存不是普通文件"))
			continue
		}
		if err := s.remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func coverCacheTrackID(name string) (uint, bool) {
	if !strings.HasSuffix(name, ".img") {
		return 0, false
	}
	parts := strings.Split(strings.TrimSuffix(name, ".img"), "-")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !decimalDigits(parts[0]) || !decimalDigits(parts[1]) {
		return 0, false
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != parts[0] || uint64(uint(id)) != id {
		return 0, false
	}
	return uint(id), true
}

func decimalDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func resourceReferenced(references []repository.TrackResourceReference, resource string, excludeID uint) bool {
	if resource == "" {
		return false
	}
	for _, reference := range references {
		if reference.ID != excludeID && (reference.URL == resource || reference.LyricPath == resource) {
			return true
		}
	}
	return false
}

func (s *ContentService) cleanupReplacedResources(current, updated model.MusicTrack) {
	if current.Source != "local" || s.uploads == nil {
		return
	}
	references, err := s.repo.TrackResourceReferences()
	if err != nil {
		slog.Error("更新后曲目资源引用扫描失败", "track_id", current.ID, "error", err)
		return
	}
	if current.URL != updated.URL && !resourceReferenced(references, current.URL, 0) {
		if err := s.uploads.DeleteMusic(current.URL); err != nil {
			slog.Error("更新后旧音频清理失败", "track_id", current.ID, "path", current.URL, "error", err)
		}
	}
	if current.LyricPath != updated.LyricPath && !resourceReferenced(references, current.LyricPath, 0) {
		if err := s.uploads.DeleteLyrics(current.LyricPath); err != nil {
			slog.Error("更新后旧歌词清理失败", "track_id", current.ID, "path", current.LyricPath, "error", err)
		}
	}
}

func (s *ContentService) cleanupDeletedResources(track model.MusicTrack, references []repository.TrackResourceReference) {
	if track.Source != "local" || s.uploads == nil {
		return
	}
	if track.LyricPath != "" && !resourceReferenced(references, track.LyricPath, track.ID) {
		if err := s.uploads.DeleteLyrics(track.LyricPath); err != nil {
			slog.Error("删除后歌词清理失败", "track_id", track.ID, "path", track.LyricPath, "error", err)
		}
	}
	if err := s.uploads.DeleteTrackCovers(track.ID); err != nil {
		slog.Error("删除后封面缓存清理失败", "track_id", track.ID, "error", err)
	}
}

func validateLocalMusicPath(root, resource string) error {
	_, err := regularLocalResourcePath(root, "music", resource)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
