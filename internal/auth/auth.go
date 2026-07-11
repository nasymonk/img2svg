package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/nasymonk/img2svg/internal/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName   = "img2svg_session"
	cookieMaxAge = 7 * 24 * 3600 // 7 天
)

type Service struct {
	db           *sql.DB // trans 的 app.db (只读)
	cookieSecret string
}

func New(transDBPath, cookieSecret string) (*Service, error) {
	db, err := sql.Open("sqlite", transDBPath+"?mode=ro&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open trans db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &Service{db: db, cookieSecret: cookieSecret}, nil
}

// VerifyPassword 验证用户名密码，返回用户信息
func (s *Service) VerifyPassword(username, password string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, is_admin, is_active FROM users WHERE username=?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.IsActive)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	if !u.IsActive {
		return nil, fmt.Errorf("账户已禁用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}
	return u, nil
}

// SetSessionCookie 设置 session cookie
// cookie 格式: token.username.signature
// signature = sha256(token + "." + username + secret)
func (s *Service) SetSessionCookie(w http.ResponseWriter, user *models.User) error {
	token := generateToken()
	payload := token + "." + user.Username
	sig := sign(payload, s.cookieSecret)

	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    payload + "." + sig,
		Path:     "/",
		Domain:   "",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // 生产环境 Nginx 做 HTTPS
	}
	http.SetCookie(w, cookie)
	return nil
}

// ValidateSession 从 cookie 验证 session，返回用户
func (s *Service) ValidateSession(r *http.Request) (*models.User, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, fmt.Errorf("未登录")
	}
	parts := strings.SplitN(cookie.Value, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("无效 session")
	}
	token, username, sig := parts[0], parts[1], parts[2]
	if token == "" || username == "" || sig == "" {
		return nil, fmt.Errorf("无效 session")
	}
	payload := token + "." + username
	if sign(payload, s.cookieSecret) != sig {
		return nil, fmt.Errorf("session 签名无效")
	}
	return &models.User{Username: username}, nil
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (s *Service) Close() error {
	return s.db.Close()
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sign(payload, secret string) string {
	h := sha256.Sum256([]byte(payload + secret))
	return hex.EncodeToString(h[:])
}
