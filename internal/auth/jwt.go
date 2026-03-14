package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/clash-proxyd/proxyd/internal/store"
)

// Claims represents JWT claims
type Claims struct {
	User string `json:"user"`
	jwt.RegisteredClaims
}

// Manager manages JWT authentication
type Manager struct {
	secret        string
	sessionTimeout int
	settingStore  *store.SettingStore
}

// NewManager creates a new auth manager
func NewManager(secret string, sessionTimeout int, settingStore *store.SettingStore) *Manager {
	return &Manager{
		secret:         secret,
		sessionTimeout: sessionTimeout,
		settingStore:   settingStore,
	}
}

// GenerateToken generates a JWT token
func (m *Manager) GenerateToken(username string) (string, int64, error) {
	expiresAt := time.Now().Add(time.Duration(m.sessionTimeout) * time.Second)

	claims := &Claims{
		User: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, expiresAt.Unix(), nil
}

// ValidateToken validates a JWT token
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Login authenticates a user
func (m *Manager) Login(username, password string) (string, int64, error) {
	// Get stored credentials
	storedUser, err := m.settingStore.Get("admin_username")
	if err != nil {
		return "", 0, fmt.Errorf("authentication failed: %w", err)
	}

	storedPass, err := m.settingStore.Get("admin_password")
	if err != nil {
		return "", 0, fmt.Errorf("authentication failed: %w", err)
	}

	// Check username
	if username != storedUser {
		return "", 0, errors.New("invalid username or password")
	}

	// If stored password is not a bcrypt hash (legacy plain-text), re-hash on successful login
	if !isBcryptHash(storedPass) {
		if password != storedPass {
			return "", 0, errors.New("invalid username or password")
		}
		// Migrate: replace plain-text password with bcrypt hash
		if hashed, err := HashPassword(password); err == nil {
			_ = m.settingStore.Set("admin_password", hashed, "Admin password (hashed)")
		}
	} else if !m.comparePassword(password, storedPass) {
		return "", 0, errors.New("invalid username or password")
	}

	// Generate token
	return m.GenerateToken(username)
}

// Logout handles logout (client-side token removal in stateless JWT)
func (m *Manager) Logout(tokenString string) error {
	// In stateless JWT, logout is handled by client deleting token
	// For token invalidation, you'd need a blacklist or Redis
	return nil
}

// comparePassword compares a plain-text password against a bcrypt hash.
func (m *Manager) comparePassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// isBcryptHash reports whether s looks like a bcrypt hash.
func isBcryptHash(s string) bool {
	return strings.HasPrefix(s, "$2")
}

// RefreshToken refreshes a JWT token
func (m *Manager) RefreshToken(tokenString string) (string, int64, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", 0, err
	}

	// Check if token is close to expiration (within 1 hour)
	if time.Until(claims.ExpiresAt.Time) > time.Hour {
		return "", 0, errors.New("token is still valid")
	}

	// Generate new token
	return m.GenerateToken(claims.User)
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// SetCredentials sets admin credentials
func (m *Manager) SetCredentials(username, password string) error {
	hashedPass, err := HashPassword(password)
	if err != nil {
		return err
	}

	if err := m.settingStore.Set("admin_username", username, "Admin username"); err != nil {
		return fmt.Errorf("failed to set username: %w", err)
	}

	if err := m.settingStore.Set("admin_password", hashedPass, "Admin password (hashed)"); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}
