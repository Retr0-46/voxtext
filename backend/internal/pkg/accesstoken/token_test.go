package accesstoken_test

import (
	"testing"
	"time"

	"backend_voxtext/internal/modules/middleware"
	"backend_voxtext/internal/pkg/accesstoken"
)

func TestIssuedTokenPassesMiddlewareVerification(t *testing.T) {
	manager, err := accesstoken.NewManager([]byte("test-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, expiresAt, err := manager.IssueAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("invalid issued token or expiry: token=%q expiresAt=%v", token, expiresAt)
	}

	var verifier middleware.TokenVerifier = manager
	userID, err := verifier.VerifyAccessToken(token)
	if err != nil || userID != 42 {
		t.Fatalf("VerifyAccessToken() = %d, %v; want 42, nil", userID, err)
	}

	otherManager, err := accesstoken.NewManager([]byte("other-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherManager.VerifyAccessToken(token); err == nil {
		t.Fatal("token signed with another secret was accepted")
	}
}
