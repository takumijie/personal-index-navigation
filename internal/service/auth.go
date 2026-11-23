package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"homepage/internal/model"
	"homepage/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const SessionDuration = 24 * time.Hour

type AuthService struct {
	repo      *repository.Repository
	limiter   *IPRateLimiter
	dummyHash []byte
}

type SessionInfo struct {
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewAuthService(repo *repository.Repository) *AuthService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("not-the-password"), bcrypt.DefaultCost)
	return &AuthService{repo: repo, limiter: NewIPRateLimiter(5, time.Minute), dummyHash: dummy}
}

func (s *AuthService) Login(username, password, ip string) (string, SessionInfo, error) {
	if !s.limiter.Allow(ip) {
		return "", SessionInfo{}, failure(http.StatusTooManyRequests, "rate_limited", "登录尝试过于频繁，请稍后重试")
	}
	user, err := s.repo.AdminByUsername(username)
	hash := s.dummyHash
	if err == nil {
		hash = []byte(user.PasswordHash)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", SessionInfo{}, err
	}
	passwordOK := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	usernameOK := subtle.ConstantTimeCompare([]byte(user.Username), []byte(username)) == 1
	if err != nil || !passwordOK || !usernameOK {
		return "", SessionInfo{}, failure(http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", SessionInfo{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	expires := time.Now().Add(SessionDuration)
	if err := s.repo.CreateSession(&model.AdminSession{TokenHash: TokenHash(token), AdminUserID: user.ID, ExpiresAt: expires}); err != nil {
		return "", SessionInfo{}, err
	}
	s.limiter.Reset(ip)
	return token, SessionInfo{Username: user.Username, ExpiresAt: expires}, nil
}

func (s *AuthService) Session(token string) (SessionInfo, error) {
	session, err := s.repo.Session(TokenHash(token))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SessionInfo{}, failure(http.StatusUnauthorized, "unauthorized", "登录状态无效或已过期")
	}
	if err != nil {
		return SessionInfo{}, err
	}
	return SessionInfo{Username: session.User.Username, ExpiresAt: session.ExpiresAt}, nil
}

func (s *AuthService) Logout(token string) error {
	if token != "" {
		return s.repo.DeleteSession(TokenHash(token))
	}
	return nil
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
