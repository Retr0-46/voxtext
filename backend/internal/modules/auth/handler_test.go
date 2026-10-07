package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend_voxtext/internal/modules/middleware"
)

type handlerServiceStub struct {
	register func(context.Context, string, string, string, string) (User, error)
	login    func(context.Context, string, string) (LoginResult, error)
	getUser  func(context.Context, int64) (User, error)
}

func (s handlerServiceStub) Register(ctx context.Context, name, surname, email, password string) (User, error) {
	return s.register(ctx, name, surname, email, password)
}

func (s handlerServiceStub) Login(ctx context.Context, email, password string) (LoginResult, error) {
	return s.login(ctx, email, password)
}

func (s handlerServiceStub) GetUserInfo(ctx context.Context, userID int64) (User, error) {
	return s.getUser(ctx, userID)
}

type tokenVerifierStub struct{}

func (tokenVerifierStub) VerifyAccessToken(raw string) (int64, error) {
	if raw != "valid-token" {
		return 0, errors.New("invalid token")
	}
	return 42, nil
}

func TestHandlerRegister(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	h := NewHandler(handlerServiceStub{
		register: func(_ context.Context, name, surname, email, password string) (User, error) {
			if name != "Ada" || surname != "Lovelace" || email != "ada@example.com" || password != "pass1234" {
				t.Fatalf("unexpected registration arguments: %q, %q, %q, %q", name, surname, email, password)
			}
			return User{Id: 42, Name: name, Surname: surname, Email: email, IsActive: true, CreatedAt: createdAt}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"Ada","usersurname":"Lovelace","email":"ada@example.com","password":"pass1234"}`))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response userResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.ID != 42 || response.Username != "Ada" || response.Usersurname != "Lovelace" || response.Email != "ada@example.com" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestHandlerLogin(t *testing.T) {
	h := NewHandler(handlerServiceStub{
		login: func(_ context.Context, email, password string) (LoginResult, error) {
			if email != "ada@example.com" || password != "pass1234" {
				t.Fatalf("unexpected login arguments: %q, %q", email, password)
			}
			return LoginResult{user: User{Id: 42, Name: "Ada"}, AccessToken: "signed-token"}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"ada@example.com","password":"pass1234"}`))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response LoginResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.ID != 42 || response.Username != "Ada" || response.AccessToken != "signed-token" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestHandlerMeUsesMiddlewareIdentity(t *testing.T) {
	h := NewHandler(handlerServiceStub{
		getUser: func(_ context.Context, userID int64) (User, error) {
			if userID != 42 {
				t.Fatalf("user ID = %d, want 42", userID)
			}
			return User{Id: 42, Name: "Ada"}, nil
		},
	})
	protected := middleware.RequireAuth(tokenVerifierStub{})(http.HandlerFunc(h.Me))
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
