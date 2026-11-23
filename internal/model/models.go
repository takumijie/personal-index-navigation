package model

import "time"

type SiteProfile struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	Description string    `json:"description"`
	Avatar      string    `json:"avatar"`
	CoverMode   string    `json:"cover_mode"`
	CoverURL    string    `json:"cover_url"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
	Tags        []string  `gorm:"-" json:"tags"`
}

func (SiteProfile) TableName() string { return "site_profile" }

type ProfileTag struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	ProfileID uint      `gorm:"uniqueIndex:idx_profile_tag" json:"-"`
	Tag       string    `gorm:"uniqueIndex:idx_profile_tag" json:"-"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type NavigationItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Icon      string    `json:"icon"`
	IconURL   string    `json:"icon_url"`
	Sort      int       `json:"sort"`
	Enabled   bool      `json:"enabled"`
	OpenMode  string    `json:"open_mode"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Device struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
	Sort        int       `json:"sort"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

type VisitorInfo struct {
	SystemName     string    `json:"system_name"`
	Distribution   string    `json:"distribution"`
	BrowserName    string    `json:"browser_name"`
	BrowserVersion string    `json:"browser_version"`
	IPAddress      string    `json:"ip_address"`
	VisitTime      time.Time `json:"visit_time"`
}

type GuestbookEntry struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SystemName     string    `json:"system_name"`
	Distribution   string    `json:"distribution"`
	BrowserName    string    `json:"browser_name"`
	BrowserVersion string    `json:"browser_version"`
	IPAddress      string    `json:"ip_address"`
	Message        string    `json:"message"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"-"`
}

type MusicTrack struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Album       string    `json:"album"`
	Cover       string    `json:"cover"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	ExternalID  string    `gorm:"index:idx_music_external" json:"external_id"`
	ObjectKey   string    `gorm:"index:idx_music_object" json:"object_key"`
	LyricPath   string    `json:"lyric_path"`
	Sort        int       `json:"sort"`
	Enabled     bool      `json:"enabled"`
	PlaylistIDs []uint    `gorm:"-" json:"playlist_ids"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

func (MusicTrack) TableName() string { return "music_tracks" }

type Playlist struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Name      string          `json:"name"`
	Sort      int             `json:"sort"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"-"`
	UpdatedAt time.Time       `json:"-"`
	Tracks    []MusicTrack    `gorm:"-" json:"tracks"`
	Items     []PlaylistTrack `gorm:"foreignKey:PlaylistID" json:"-"`
}

type PlaylistTrack struct {
	PlaylistID uint      `gorm:"primaryKey" json:"-"`
	TrackID    uint      `gorm:"primaryKey" json:"-"`
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"-"`
}

type MusicProviderConfig struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	Provider  string    `gorm:"uniqueIndex" json:"-"`
	Data      string    `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type SiteSettings struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	Data      string    `json:"-"`
	Version   uint64    `gorm:"not null;default:1" json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type AdminUser struct {
	ID           uint      `gorm:"primaryKey" json:"-"`
	Username     string    `gorm:"uniqueIndex" json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}

type AdminSession struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	TokenHash   string    `gorm:"uniqueIndex" json:"-"`
	AdminUserID uint      `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	User        AdminUser `gorm:"foreignKey:AdminUserID" json:"-"`
}

type Background struct {
	DayURL     string `json:"day_url"`
	NightURL   string `json:"night_url"`
	DayStart   string `json:"day_start"`
	NightStart string `json:"night_start"`
	Timezone   string `json:"timezone"`
	DurationMS int    `json:"duration_ms"`
}

type CardSettings struct {
	Profile    bool `json:"profile"`
	Navigation bool `json:"navigation"`
	Devices    bool `json:"devices"`
	Terminal   bool `json:"terminal"`
	Clock      bool `json:"clock"`
}

type LoaderSettings struct {
	Theme             string `json:"theme"`
	SoundEnabled      bool   `json:"sound_enabled"`
	WatermarkTitle    string `json:"watermark_title"`
	WatermarkSubtitle string `json:"watermark_subtitle"`
	WatermarkCaption  string `json:"watermark_caption"`
	BadgeImage        string `json:"badge_image"`
}

type TerminalSettings struct {
	FastfetchLoadingMS int `json:"fastfetch_loading_ms"`
}

type MusicSettings struct {
	LyricsEnabled bool `json:"lyrics_enabled"`
	PlaylistID    uint `json:"playlist_id"`
}

type Settings struct {
	Version              uint64           `json:"version"`
	SiteTitle            string           `json:"site_title"`
	Timezone             string           `json:"timezone"`
	TimeFormat           string           `json:"time_format"`
	ClockLabel           string           `json:"clock_label"`
	Background           Background       `json:"background"`
	Cards                CardSettings     `json:"cards"`
	Loader               LoaderSettings   `json:"loader"`
	Terminal             TerminalSettings `json:"terminal"`
	Music                MusicSettings    `json:"music"`
	GuestbookAutoPublish bool             `json:"guestbook_auto_publish"`
	GuestbookMaxLength   int              `json:"guestbook_max_length"`
}
