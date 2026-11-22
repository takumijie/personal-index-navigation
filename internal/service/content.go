package service

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"homepage/internal/model"
	"homepage/internal/repository"

	"gorm.io/gorm"
)

type ContentService struct {
	repo            *repository.Repository
	guestbookLimit  *IPRateLimiter
}

func NewContentService(repo *repository.Repository) *ContentService {
	return &ContentService{repo: repo, guestbookLimit: NewIPRateLimiter(5, time.Minute)}
}

func (s *ContentService) Profile() (model.SiteProfile, error) { return s.repo.Profile() }

func (s *ContentService) UpdateProfile(profile model.SiteProfile) error {
	if err := validateProfile(profile); err != nil {
		return err
	}
	return s.repo.UpdateProfile(profile)
}

func (s *ContentService) Navigation(enabledOnly bool) ([]model.NavigationItem, error) {
	return s.repo.Navigation(enabledOnly)
}

func (s *ContentService) CreateNavigation(item model.NavigationItem) (model.NavigationItem, error) {
	item.ID = 0
	if err := validateNavigation(item); err != nil {
		return item, err
	}
	return item, s.repo.SaveNavigation(&item)
}

func (s *ContentService) UpdateNavigation(id uint, item model.NavigationItem) (model.NavigationItem, error) {
	current, err := s.repo.NavigationByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, notFound("导航不存在")
	}
	if err != nil {
		return item, err
	}
	if err := validateNavigation(item); err != nil {
		return item, err
	}
	item.ID, item.CreatedAt = current.ID, current.CreatedAt
	return item, s.repo.SaveNavigation(&item)
}

func (s *ContentService) DeleteNavigation(id uint) error {
	if _, err := s.repo.NavigationByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound("导航不存在")
	} else if err != nil {
		return err
	}
	return s.repo.DeleteNavigation(id)
}

func (s *ContentService) Devices(enabledOnly bool) ([]model.Device, error) {
	return s.repo.Devices(enabledOnly)
}

func (s *ContentService) CreateDevice(item model.Device) (model.Device, error) {
	item.ID = 0
	if err := validateDevice(item); err != nil {
		return item, err
	}
	return item, s.repo.SaveDevice(&item)
}

func (s *ContentService) UpdateDevice(id uint, item model.Device) (model.Device, error) {
	current, err := s.repo.DeviceByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, notFound("设备不存在")
	}
	if err != nil {
		return item, err
	}
	if err := validateDevice(item); err != nil {
		return item, err
	}
	item.ID, item.CreatedAt = current.ID, current.CreatedAt
	return item, s.repo.SaveDevice(&item)
}

func (s *ContentService) DeleteDevice(id uint) error {
	if _, err := s.repo.DeviceByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound("设备不存在")
	} else if err != nil {
		return err
	}
	return s.repo.DeleteDevice(id)
}

func (s *ContentService) Visitor(input GuestbookInput, ip string) (model.VisitorInfo, error) {
	for _, value := range []string{input.SystemName, input.Distribution, input.BrowserName, input.BrowserVersion} {
		if !validLength(value, 100) || !plainText(value, false) {
			return model.VisitorInfo{}, invalid("访客系统信息无效")
		}
	}
	return model.VisitorInfo{
		SystemName: input.SystemName, Distribution: input.Distribution,
		BrowserName: input.BrowserName, BrowserVersion: input.BrowserVersion,
		IPAddress: ip, VisitTime: time.Now(),
	}, nil
}

func (s *ContentService) Guestbook(limit int) ([]model.GuestbookEntry, error) {
	return s.repo.Guestbook("published", limit)
}

func (s *ContentService) AdminGuestbook() ([]model.GuestbookEntry, error) {
	return s.repo.Guestbook("", 0)
}

type GuestbookInput struct {
	SystemName     string `json:"system_name"`
	Distribution   string `json:"distribution"`
	BrowserName    string `json:"browser_name"`
	BrowserVersion string `json:"browser_version"`
	Message        string `json:"message"`
}

func (s *ContentService) CreateGuestbook(input GuestbookInput, ip string) (model.GuestbookEntry, error) {
	var entry model.GuestbookEntry
	if !s.guestbookLimit.Allow(ip) {
		return entry, failure(http.StatusTooManyRequests, "rate_limited", "留言过于频繁，请稍后重试")
	}
	settings, err := s.Settings()
	if err != nil {
		return entry, err
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" || !plainText(input.Message, true) || utf8.RuneCountInString(input.Message) > settings.GuestbookMaxLength {
		return entry, invalid("留言为空、超长或包含控制字符")
	}
	for _, value := range []string{input.SystemName, input.Distribution, input.BrowserName, input.BrowserVersion} {
		if !validLength(value, 100) || !plainText(value, false) {
			return entry, invalid("访客系统信息无效")
		}
	}
	status := "pending"
	if settings.GuestbookAutoPublish {
		status = "published"
	}
	entry = model.GuestbookEntry{
		SystemName: input.SystemName, Distribution: input.Distribution,
		BrowserName: input.BrowserName, BrowserVersion: input.BrowserVersion,
		IPAddress: ip, Message: input.Message, Status: status,
	}
	return entry, s.repo.SaveGuestbook(&entry)
}

func (s *ContentService) UpdateGuestbookStatus(id uint, status string) (model.GuestbookEntry, error) {
	entry, err := s.repo.GuestbookByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entry, notFound("留言不存在")
	}
	if err != nil {
		return entry, err
	}
	if !oneOf(status, "pending", "published", "rejected") {
		return entry, invalid("留言状态无效")
	}
	entry.Status = status
	return entry, s.repo.SaveGuestbook(&entry)
}

func (s *ContentService) DeleteGuestbook(id uint) error {
	if _, err := s.repo.GuestbookByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound("留言不存在")
	} else if err != nil {
		return err
	}
	return s.repo.DeleteGuestbook(id)
}

func (s *ContentService) Settings() (model.Settings, error) { return s.repo.Settings() }

func (s *ContentService) UpdateSettings(settings model.Settings) error {
	if err := validateSettings(settings); err != nil {
		return err
	}
	return s.repo.SaveSettings(settings)
}

func (s *ContentService) Tracks(enabledOnly bool) ([]model.MusicTrack, error) {
	return s.repo.Tracks(enabledOnly)
}

func (s *ContentService) CreateTrack(item model.MusicTrack) (model.MusicTrack, error) {
	item.ID = 0
	if err := validateTrack(item); err != nil {
		return item, err
	}
	return item, s.repo.SaveTrack(&item)
}

func (s *ContentService) UpdateTrack(id uint, item model.MusicTrack) (model.MusicTrack, error) {
	current, err := s.repo.TrackByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, notFound("曲目不存在")
	}
	if err != nil {
		return item, err
	}
	if err := validateTrack(item); err != nil {
		return item, err
	}
	item.ID, item.CreatedAt = current.ID, current.CreatedAt
	return item, s.repo.SaveTrack(&item)
}

func (s *ContentService) DeleteTrack(id uint) error {
	if _, err := s.repo.TrackByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound("曲目不存在")
	} else if err != nil {
		return err
	}
	return s.repo.DeleteTrack(id)
}

func (s *ContentService) Playlist() (model.Playlist, error) { return s.repo.Playlist() }
