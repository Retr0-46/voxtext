package auth

import (
	"context"
	"strings"
)

type RepoInterface interface {
	CreateUser(ctx context.Context, name, surname, email, passwordHash string) (User, error)
	GetUser(ctx context.Context, identifier any) (User, error)
	SetBalance(ctx context.Context, userID int64, balance int64) (User, error)
}

type Service struct {
	repo         RepoInterface
	tokenManager TokenManagerInterface
}

func NewService(repository RepoInterface, tokenManager TokenManagerInterface) *Service {
	return &Service{repo: repository, tokenManager: tokenManager}
}

func (s *Service) Register(ctx context.Context, username, usersurname, email, password string) (User, error) {

	name := strings.TrimSpace(username)
	surname := strings.TrimSpace(usersurname)

	if len(name) < 1 || len(surname) < 1 {
		return User{}, ErrEmptyUsername
	}

	if len([]rune(password)) < 4 {
		return User{}, ErrTooShortPassword
	}

	if len([]byte(password)) > 72 {
		return User{}, ErrTooLongPassword
	}

	hashPassword, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	user, err := s.repo.CreateUser(ctx, name, surname, email, hashPassword)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.repo.GetUser(ctx, email)

	if err != nil {
		return LoginResult{}, err
	}

	if ComparePassword(user.PasswordHash, password) != nil {
		return LoginResult{}, ErrPasswordMismatch
	}

	token, expiresAt, err := s.tokenManager.IssueAccessToken(user.Id)

	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{user, token, expiresAt}, nil
}

func (s *Service) GetUserInfo(ctx context.Context, userID int64) (User, error) {
	user, err := s.repo.GetUser(ctx, userID)

	if err != nil {
		return User{}, err
	}

	return user, nil
}
