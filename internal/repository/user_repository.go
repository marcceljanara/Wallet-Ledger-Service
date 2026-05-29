package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository/sqlc"
)

type UserRepository interface {
	Create(ctx context.Context, tx pgx.Tx, email, passwordHash string, role model.UserRole) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindAll(ctx context.Context, limit, offset int) ([]model.UserWithWallet, int, error)
}

type userRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *userRepository) getQueries(tx pgx.Tx) *sqlc.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *userRepository) Create(ctx context.Context, tx pgx.Tx, email, passwordHash string, role model.UserRole) (*model.User, error) {
	q := r.getQueries(tx)
	u, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         string(role),
	})
	if err != nil {
		return nil, fmt.Errorf("create user query failed: %w", err)
	}
	domainUser := mapUser(u)
	return &domainUser, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := r.queries.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email query failed: %w", err)
	}
	domainUser := mapUser(u)
	return &domainUser, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	u, err := r.queries.FindUserByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id query failed: %w", err)
	}
	domainUser := mapUser(u)
	return &domainUser, nil
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int) ([]model.UserWithWallet, int, error) {
	users, err := r.queries.FindAllUsers(ctx, sqlc.FindAllUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("find all users query failed: %w", err)
	}

	total, err := r.queries.CountAllUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count all users query failed: %w", err)
	}

	domainUsers := make([]model.UserWithWallet, len(users))
	for i, u := range users {
		domainUsers[i] = mapFindAllUsersRow(u)
	}

	return domainUsers, int(total), nil
}

func mapFindAllUsersRow(u sqlc.FindAllUsersRow) model.UserWithWallet {
	var id uuid.UUID
	if u.ID.Valid {
		id = u.ID.Bytes
	}
	var createdAt time.Time
	if u.CreatedAt.Valid {
		createdAt = u.CreatedAt.Time
	}
	var updatedAt time.Time
	if u.UpdatedAt.Valid {
		updatedAt = u.UpdatedAt.Time
	}
	walletID := ""
	if u.WalletID.Valid {
		walletID = u.WalletID.String
	}
	return model.UserWithWallet{
		User: model.User{
			ID:        id,
			Email:     u.Email,
			Role:      model.UserRole(u.Role),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		WalletID:      walletID,
		WalletBalance: numericToDecimal(u.WalletBalance),
	}
}

func mapUser(u sqlc.User) model.User {
	var id uuid.UUID
	if u.ID.Valid {
		id = u.ID.Bytes
	}
	var createdAt time.Time
	if u.CreatedAt.Valid {
		createdAt = u.CreatedAt.Time
	}
	var updatedAt time.Time
	if u.UpdatedAt.Valid {
		updatedAt = u.UpdatedAt.Time
	}
	return model.User{
		ID:           id,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         model.UserRole(u.Role),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
