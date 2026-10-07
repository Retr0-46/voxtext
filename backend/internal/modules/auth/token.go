package auth

import "time"

type TokenManagerInterface interface {
	IssueAccessToken(userID int64) (value string, expiresAt time.Time, err error)
}
