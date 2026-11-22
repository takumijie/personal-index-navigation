package repository

import (
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
	return items, tx.Find(&items).Error
}

func (r *Repository) TrackByID(id uint) (model.MusicTrack, error) {
	var item model.MusicTrack
	return item, r.db.First(&item, id).Error
}

func (r *Repository) SaveTrack(item *model.MusicTrack) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(item).Error; err != nil {
			return err
		}
		return syncPlaylist(tx)
	})
}

func (r *Repository) DeleteTrack(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.MusicTrack{}, id).Error; err != nil {
			return err
		}
		return syncPlaylist(tx)
	})
}

func (r *Repository) Playlist() (model.Playlist, error) {
	var playlist model.Playlist
	if err := r.db.Where("enabled = ?", true).Order("sort, id").First(&playlist).Error; err != nil {
		return playlist, err
	}
	playlist.Tracks = make([]model.MusicTrack, 0)
	err := r.db.Model(&model.MusicTrack{}).
		Joins("JOIN playlist_tracks ON playlist_tracks.track_id = music_tracks.id").
		Where("playlist_tracks.playlist_id = ? AND music_tracks.enabled = ?", playlist.ID, true).
		Order("playlist_tracks.sort, music_tracks.id").Find(&playlist.Tracks).Error
	return playlist, err
}

func syncPlaylist(tx *gorm.DB) error {
	var playlist model.Playlist
	err := tx.Order("sort, id").First(&playlist).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		playlist = model.Playlist{Name: "默认播放列表", Enabled: true}
		if err := tx.Create(&playlist).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := tx.Where("playlist_id = ?", playlist.ID).Delete(&model.PlaylistTrack{}).Error; err != nil {
		return err
	}
	var tracks []model.MusicTrack
	if err := tx.Where("enabled = ?", true).Order("sort, id").Find(&tracks).Error; err != nil {
		return err
	}
	for _, track := range tracks {
		if err := tx.Create(&model.PlaylistTrack{PlaylistID: playlist.ID, TrackID: track.ID, Sort: track.Sort}).Error; err != nil {
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
