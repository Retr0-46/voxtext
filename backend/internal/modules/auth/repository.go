package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, name, surname, email, passwordHash string) (User, error) {
	var user User

	row := r.pool.QueryRow(ctx, `INSERT INTO users (username, usersurname, email, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, username, usersurname, email, password_hash, is_active, balance, created_at, updated_at`, name, surname, email, passwordHash)
	err := row.Scan(&user.Id, &user.Name, &user.Surname, &user.Email, &user.PasswordHash, &user.IsActive, &user.balance, &user.CreatedAt, &user.UpdatedAt)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23505" && (pgErr.ConstraintName == "users_username_unique" || pgErr.ConstraintName == "users_email_unique") {
		return User{}, ErrUserAlreadyExist
	}

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (r *Repository) GetUser(ctx context.Context, identifier any) (User, error) {
	switch t := identifier.(type) {
	case string:
		return r.getUserByEmail(ctx, t)
	case int:
		return r.getUserByID(ctx, int64(t))
	case int64:
		return r.getUserByID(ctx, t)
	default:
		return User{}, ErrWrongIdentificator
	}
}

func (r *Repository) SetBalance(ctx context.Context, userID int64, balance int64) (User, error) {
	var user User

	row := r.pool.QueryRow(ctx, `UPDATE users SET balance = $1, updated_at = NOW() WHERE id = $2 RETURNING id, username, usersurname, email, password_hash, is_active, balance, created_at, updated_at`, balance, userID)
	err := row.Scan(&user.Id, &user.Name, &user.Surname, &user.Email, &user.PasswordHash, &user.IsActive, &user.balance, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (r *Repository) getUserByID(ctx context.Context, id int64) (User, error) {
	var user User

	row := r.pool.QueryRow(ctx, `SELECT id, username, usersurname, email, password_hash, is_active, balance, created_at, updated_at FROM users WHERE id = $1`, id)
	err := row.Scan(&user.Id, &user.Name, &user.Surname, &user.Email, &user.PasswordHash, &user.IsActive, &user.balance, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (r *Repository) getUserByEmail(ctx context.Context, name string) (User, error) {
	var user User

	row := r.pool.QueryRow(ctx, `SELECT id, username, usersurname, email, password_hash, is_active, balance, created_at, updated_at FROM users WHERE email = $1`, name)
	err := row.Scan(&user.Id, &user.Name, &user.Surname, &user.Email, &user.PasswordHash, &user.IsActive, &user.balance, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	if err != nil {
		return User{}, err
	}

	return user, nil
}
