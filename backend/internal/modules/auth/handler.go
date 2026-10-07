package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"backend_voxtext/internal/modules/middleware"
)

type ServiceInterface interface {
	Register(ctx context.Context, username, usersurname, email, password string) (User, error)
	Login(ctx context.Context, email, password string) (LoginResult, error)
	GetUserInfo(ctx context.Context, userID int64) (User, error)
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(serv ServiceInterface) *Handler {
	return &Handler{service: serv}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	user, err := h.service.Register(r.Context(), req.Username, req.Usersurname, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExist) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrEmptyUsername) || errors.Is(err, ErrTooShortPassword) || errors.Is(err, ErrTooLongPassword) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUserResponse(user))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	res, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrPasswordMismatch) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		ID:          res.user.Id,
		Username:    res.user.Name,
		AccessToken: res.AccessToken,
		ExpiresAt:   res.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetUserInfo(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUserResponse(user))
}

func newUserResponse(user User) userResponse {
	return userResponse{
		ID:          user.Id,
		Username:    user.Name,
		Usersurname: user.Surname,
		Email:       user.Email,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt,
	}
}
