package repository

import (
	"encoding/json"
	"errors"
	"time"

	"homepage/internal/model"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Profile() (model.SiteProfile, error) {
	var profile model.SiteProfile
	if err := r.db.Order("id").First(&profile).Error; err != nil {
		return profile, err
	}
	var tags []model.ProfileTag
	if err := r.db.Where("profile_id = ?", profile.ID).Order("sort, id").Find(&tags).Error; err != nil {
		return profile, err
	}
	profile.Tags = make([]string, len(tags))
	for i, tag := range tags {
		profile.Tags[i] = tag.Tag
	}
	return profile, nil
}

func (r *Repository) UpdateProfile(profile model.SiteProfile) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var current model.SiteProfile
		if err := tx.Order("id").First(&current).Error; err != nil {
			return err
		}
		profile.ID = current.ID
		if err := tx.Model(&current).Updates(map[string]any{
			"name": profile.Name, "username": profile.Username, "description": profile.Description,
			"avatar": profile.Avatar, "cover_mode": profile.CoverMode, "cover_url": profile.CoverURL,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("profile_id = ?", current.ID).Delete(&model.ProfileTag{}).Error; err != nil {
			return err
		}
		for i, tag := range profile.Tags {
			if err := tx.Create(&model.ProfileTag{ProfileID: current.ID, Tag: tag, Sort: i}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) Navigation(enabledOnly bool) ([]model.NavigationItem, error) {
	items := make([]model.NavigationItem, 0)
	tx := r.db.Order("sort, id")
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	return items, tx.Find(&items).Error
}

func (r *Repository) NavigationByID(id uint) (model.NavigationItem, error) {
	var item model.NavigationItem
	return item, r.db.First(&item, id).Error
}

func (r *Repository) SaveNavigation(item *model.NavigationItem) error {
	return r.db.Save(item).Error
}

func (r *Repository) DeleteNavigation(id uint) error {
	return r.db.Delete(&model.NavigationItem{}, id).Error
}

func (r *Repository) Devices(enabledOnly bool) ([]model.Device, error) {
	items := make([]model.Device, 0)
	tx := r.db.Order("sort, id")
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	return items, tx.Find(&items).Error
}

func (r *Repository) DeviceByID(id uint) (model.Device, error) {
	var item model.Device
	return item, r.db.First(&item, id).Error
}

func (r *Repository) SaveDevice(item *model.Device) error { return r.db.Save(item).Error }

func (r *Repository) DeleteDevice(id uint) error { return r.db.Delete(&model.Device{}, id).Error }

func (r *Repository) Guestbook(status string, limit int) ([]model.GuestbookEntry, error) {
	items := make([]model.GuestbookEntry, 0)
	tx := r.db.Order("created_at DESC, id DESC")
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&items).Error; err != nil {
		return items, err
	}
	if status == "published" {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
	}
	return items, nil
}

func (r *Repository) GuestbookByID(id uint) (model.GuestbookEntry, error) {
	var item model.GuestbookEntry
	return item, r.db.First(&item, id).Error
}

func (r *Repository) SaveGuestbook(item *model.GuestbookEntry) error {
	return r.db.Save(item).Error
}

func (r *Repository) DeleteGuestbook(id uint) error {
	return r.db.Delete(&model.GuestbookEntry{}, id).Error
}

func (r *Repository) Settings() (model.Settings, error) {
	var row model.SiteSettings
	if err := r.db.Order("id").First(&row).Error; err != nil {
		return model.Settings{}, err
	}
	return decodeSettings(row.Data)
}

func (r *Repository) SaveSettings(settings model.Settings) error {
	data, err := encodeSettings(settings)
	if err != nil {
		return err
	}
	var row model.SiteSettings
	if err := r.db.Order("id").First(&row).Error; err != nil {
		return err
	}
	return r.db.Model(&row).Updates(map[string]any{"data": data, "updated_at": time.Now()}).Error
}

func (r *Repository) Tracks(enabledOnly bool) ([]model.MusicTrack, error) {
	items := make([]model.MusicTrack, 0)
	tx := r.db.Order("sort, id")
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	if err := tx.Find(&items).Error; err != nil {
		return items, err
	}
	return items, r.loadTrackPlaylistIDs(items)
}

func (r *Repository) TrackByID(id uint) (model.MusicTrack, error) {
	var item model.MusicTrack
	if err := r.db.First(&item, id).Error; err != nil {
		return item, err
	}
	items := []model.MusicTrack{item}
	if err := r.loadTrackPlaylistIDs(items); err != nil {
		return item, err
	}
	return items[0], nil
}

func (r *Repository) PublicTrackByID(id uint) (model.MusicTrack, error) {
	var item model.MusicTrack
	err := r.db.Model(&model.MusicTrack{}).
		Joins("JOIN playlist_tracks ON playlist_tracks.track_id = music_tracks.id").
		Joins("JOIN playlists ON playlists.id = playlist_tracks.playlist_id").
		Where("music_tracks.id = ? AND music_tracks.enabled = ? AND playlists.enabled = ?", id, true, true).
		First(&item).Error
	return item, err
}

func (r *Repository) SaveTrack(item *model.MusicTrack) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		isNew := item.ID == 0
		if err := tx.Save(item).Error; err != nil {
			return err
		}
		if item.PlaylistIDs == nil {
			if !isNew {
				return nil
			}
			playlist, err := defaultPlaylist(tx)
			if err != nil {
				return err
			}
			item.PlaylistIDs = []uint{playlist.ID}
		}
		if err := tx.Where("track_id = ?", item.ID).Delete(&model.PlaylistTrack{}).Error; err != nil {
			return err
		}
		seen := make(map[uint]struct{}, len(item.PlaylistIDs))
		for _, playlistID := range item.PlaylistIDs {
			if _, ok := seen[playlistID]; ok {
				continue
			}
			seen[playlistID] = struct{}{}
			if err := tx.Create(&model.PlaylistTrack{PlaylistID: playlistID, TrackID: item.ID, Sort: item.Sort}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) DeleteTrack(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("track_id = ?", id).Delete(&model.PlaylistTrack{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.MusicTrack{}, id).Error
	})
}

func (r *Repository) Playlist() (model.Playlist, error) {
	var playlist model.Playlist
	if err := r.db.Where("enabled = ?", true).Order("sort, id").First(&playlist).Error; err != nil {
		return playlist, err
	}
	return r.PlaylistByID(playlist.ID, true)
}

func (r *Repository) Playlists(enabledOnly, withTracks bool) ([]model.Playlist, error) {
	items := make([]model.Playlist, 0)
	tx := r.db.Order("sort, id")
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	if err := tx.Find(&items).Error; err != nil {
		return items, err
	}
	if withTracks {
		for i := range items {
			item, err := r.PlaylistByID(items[i].ID, enabledOnly)
			if err != nil {
				return items, err
			}
			items[i] = item
		}
	}
	return items, nil
}

func (r *Repository) PlaylistByID(id uint, enabledOnly bool) (model.Playlist, error) {
	var playlist model.Playlist
	tx := r.db
	if enabledOnly {
		tx = tx.Where("enabled = ?", true)
	}
	if err := tx.First(&playlist, id).Error; err != nil {
		return playlist, err
	}
	playlist.Tracks = make([]model.MusicTrack, 0)
	trackQuery := r.db.Model(&model.MusicTrack{}).
		Joins("JOIN playlist_tracks ON playlist_tracks.track_id = music_tracks.id").
		Where("playlist_tracks.playlist_id = ?", playlist.ID)
	if enabledOnly {
		trackQuery = trackQuery.Where("music_tracks.enabled = ?", true)
	}
	err := trackQuery.Order("playlist_tracks.sort, music_tracks.id").Find(&playlist.Tracks).Error
	return playlist, err
}

func (r *Repository) SavePlaylist(item *model.Playlist) error { return r.db.Save(item).Error }

func (r *Repository) DeletePlaylist(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("playlist_id = ?", id).Delete(&model.PlaylistTrack{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Playlist{}, id).Error
	})
}

func (r *Repository) SetPlaylistTracks(id uint, trackIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var playlist model.Playlist
		if err := tx.First(&playlist, id).Error; err != nil {
			return err
		}
		seen := make(map[uint]struct{}, len(trackIDs))
		unique := make([]uint, 0, len(trackIDs))
		for _, trackID := range trackIDs {
			if _, ok := seen[trackID]; ok {
				continue
			}
			seen[trackID] = struct{}{}
			unique = append(unique, trackID)
		}
		if len(unique) > 0 {
			var count int64
			if err := tx.Model(&model.MusicTrack{}).Where("id IN ?", unique).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(unique)) {
				return gorm.ErrRecordNotFound
			}
		}
		if err := tx.Where("playlist_id = ?", id).Delete(&model.PlaylistTrack{}).Error; err != nil {
			return err
		}
		for sort, trackID := range unique {
			if err := tx.Create(&model.PlaylistTrack{PlaylistID: id, TrackID: trackID, Sort: sort}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) ImportPlaylist(playlist model.Playlist, tracks []model.MusicTrack) (model.Playlist, int, error) {
	duplicates := 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&playlist).Error; err != nil {
			return err
		}
		seen := make(map[uint]struct{}, len(tracks))
		for _, candidate := range tracks {
			var track model.MusicTrack
			query := tx.Where("source = ?", candidate.Source)
			if candidate.Source == "netease" {
				query = query.Where("external_id = ?", candidate.ExternalID)
			} else {
				query = query.Where("object_key = ?", candidate.ObjectKey)
			}
			err := query.First(&track).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				track = candidate
				track.ID = 0
				track.PlaylistIDs = nil
				if err := tx.Create(&track).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				duplicates++
			}
			if _, ok := seen[track.ID]; ok {
				continue
			}
			seen[track.ID] = struct{}{}
			if err := tx.Create(&model.PlaylistTrack{PlaylistID: playlist.ID, TrackID: track.ID, Sort: len(playlist.Tracks)}).Error; err != nil {
				return err
			}
			playlist.Tracks = append(playlist.Tracks, track)
		}
		return nil
	})
	return playlist, duplicates, err
}

func (r *Repository) ProviderConfig(provider string, target any) error {
	var row model.MusicProviderConfig
	err := r.db.Where("provider = ?", provider).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(row.Data), target)
}

func (r *Repository) SaveProviderConfig(provider string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var row model.MusicProviderConfig
	err = r.db.Where("provider = ?", provider).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.MusicProviderConfig{Provider: provider, Data: string(data)}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&row).Updates(map[string]any{"data": string(data), "updated_at": time.Now()}).Error
}

func defaultPlaylist(tx *gorm.DB) (model.Playlist, error) {
	var playlist model.Playlist
	err := tx.Order("sort, id").First(&playlist).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		playlist = model.Playlist{Name: "默认播放列表", Enabled: true}
		err = tx.Create(&playlist).Error
	}
	return playlist, err
}

func (r *Repository) loadTrackPlaylistIDs(items []model.MusicTrack) error {
	for i := range items {
		items[i].PlaylistIDs = make([]uint, 0)
		if err := r.db.Model(&model.PlaylistTrack{}).Where("track_id = ?", items[i].ID).Order("playlist_id").Pluck("playlist_id", &items[i].PlaylistIDs).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) AdminByUsername(username string) (model.AdminUser, error) {
	var user model.AdminUser
	return user, r.db.Where("username = ?", username).First(&user).Error
}

func (r *Repository) CreateSession(session *model.AdminSession) error {
	if err := r.db.Where("expires_at <= ?", time.Now()).Delete(&model.AdminSession{}).Error; err != nil {
		return err
	}
	return r.db.Create(session).Error
}

func (r *Repository) Session(tokenHash string) (model.AdminSession, error) {
	var session model.AdminSession
	err := r.db.Preload("User").Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&session).Error
	return session, err
}

func (r *Repository) DeleteSession(tokenHash string) error {
	return r.db.Where("token_hash = ?", tokenHash).Delete(&model.AdminSession{}).Error
}
