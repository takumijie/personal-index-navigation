package handler

import (
	"errors"
	"net/http"
	"strconv"

	"homepage/internal/middleware"
	"homepage/internal/model"
	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

const sessionCookie = "homepage_session"

func (a *API) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &input) {
		return
	}
	if len(input.Username) > 100 || len(input.Password) > 200 {
		fail(c, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
		return
	}
	token, session, err := a.auth.Login(input.Username, input.Password, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}
	a.setSessionCookie(c, token, int(service.SessionDuration.Seconds()))
	success(c, session)
}

func (a *API) Logout(c *gin.Context) {
	token, _ := c.Cookie(sessionCookie)
	if err := a.auth.Logout(token); err != nil {
		handleError(c, err)
		return
	}
	a.setSessionCookie(c, "", -1)
	success(c, gin.H{"logged_out": true})
}

func (a *API) Session(c *gin.Context) {
	session := c.MustGet(middleware.SessionKey).(service.SessionInfo)
	success(c, session)
}

func (a *API) setSessionCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, token, maxAge, "/", "", a.cookieSecure, true)
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "资源 ID 无效")
		return 0, false
	}
	return uint(id), true
}

func (a *API) UpdateProfile(c *gin.Context) {
	var profile model.SiteProfile
	if !bindJSON(c, &profile) {
		return
	}
	if err := a.content.UpdateProfile(profile); err != nil {
		handleError(c, err)
		return
	}
	updated, err := a.content.Profile()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) AdminNavigation(c *gin.Context) {
	items, err := a.content.Navigation(false)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) AdminDevices(c *gin.Context) {
	items, err := a.content.Devices(false)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) CreateNavigation(c *gin.Context) {
	var item model.NavigationItem
	if !bindJSON(c, &item) {
		return
	}
	created, err := a.content.CreateNavigation(item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, created)
}

func (a *API) UpdateNavigation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item model.NavigationItem
	if !bindJSON(c, &item) {
		return
	}
	updated, err := a.content.UpdateNavigation(id, item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) DeleteNavigation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.content.DeleteNavigation(id); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (a *API) CreateDevice(c *gin.Context) {
	var item model.Device
	if !bindJSON(c, &item) {
		return
	}
	created, err := a.content.CreateDevice(item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, created)
}

func (a *API) UpdateDevice(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item model.Device
	if !bindJSON(c, &item) {
		return
	}
	updated, err := a.content.UpdateDevice(id, item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) DeleteDevice(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.content.DeleteDevice(id); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (a *API) AdminGuestbook(c *gin.Context) {
	items, err := a.content.AdminGuestbook()
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) UpdateGuestbook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if !bindJSON(c, &input) {
		return
	}
	entry, err := a.content.UpdateGuestbookStatus(id, input.Status)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, entry)
}

func (a *API) DeleteGuestbook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.content.DeleteGuestbook(id); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (a *API) UpdateSettings(c *gin.Context) {
	var settings model.Settings
	if !bindJSON(c, &settings) {
		return
	}
	updated, err := a.content.UpdateSettings(settings)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) AdminTracks(c *gin.Context) {
	items, err := a.content.Tracks(false)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, items)
}

func (a *API) CreateTrack(c *gin.Context) {
	var item model.MusicTrack
	if !bindJSON(c, &item) {
		return
	}
	created, err := a.content.CreateTrack(item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, created)
}

func (a *API) UpdateTrack(c *gin.Context) {
	var item model.MusicTrack
	if !bindJSON(c, &item) {
		return
	}
	if item.ID == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "曲目 ID 无效")
		return
	}
	updated, err := a.content.UpdateTrack(item.ID, item)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, updated)
}

func (a *API) DeleteTrack(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if !bindJSON(c, &input) {
		return
	}
	if input.ID == 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "曲目 ID 无效")
		return
	}
	if err := a.content.DeleteTrack(input.ID); err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (a *API) UploadImage(c *gin.Context) { a.upload(c, "images") }

func (a *API) UploadMusic(c *gin.Context) { a.upload(c, "music") }

func (a *API) UploadLyrics(c *gin.Context) { a.upload(c, "lyrics") }

func (a *API) upload(c *gin.Context, kind string) {
	header, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			fail(c, http.StatusRequestEntityTooLarge, "request_too_large", "请求体超过大小限制")
			return
		}
		fail(c, http.StatusBadRequest, "invalid_file", "请使用 multipart 的 file 字段上传文件")
		return
	}
	url, err := a.uploads.Save(kind, header)
	if err != nil {
		handleError(c, err)
		return
	}
	success(c, gin.H{"url": url})
}
