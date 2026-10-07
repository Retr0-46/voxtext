package accesstoken

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer   = "voxtext"
	audience = "voxtext-api"
)

var (
	ErrInvalidToken   = errors.New("invalid access token")
	ErrInvalidSubject = errors.New("invalid token subject")
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret []byte, ttl time.Duration) (*Manager, error) {
	if len(secret) == 0 {
		return nil, errors.New("JWT secret must not be empty")
	}
	if ttl <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &Manager{secret: append([]byte(nil), secret...), ttl: ttl}, nil
}

func (m *Manager) IssueAccessToken(userID int64) (string, time.Time, error) {
	if userID <= 0 {
		return "", time.Time{}, ErrInvalidSubject
	}

	now := time.Now()
	expiresAt := now.Add(m.ttl)
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return value, expiresAt, nil
}

func (m *Manager) VerifyAccessToken(raw string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(_ *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, ErrInvalidSubject
	}
	return userID, nil
}
