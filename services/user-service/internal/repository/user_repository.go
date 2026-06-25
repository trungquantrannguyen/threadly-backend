package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound  = errors.New("User not found")
	ErrDuplicateUser = errors.New("Email or username already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *dbmodel.User) error
	FindByID(ctx context.Context, id string) (*dbmodel.User, error)
	FindByEmail(ctx context.Context, email string) (*dbmodel.User, error)
	FindByUsername(ctx context.Context, username string) (*dbmodel.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, user *dbmodel.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateUser
		}
		return err
	}

	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*dbmodel.User, error) {
	var user dbmodel.User

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*dbmodel.User, error) {
	var user dbmodel.User

	err := r.db.WithContext(ctx).Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*dbmodel.User, error) {
	var user dbmodel.User

	err := r.db.WithContext(ctx).Where("LOWER(username) = LOWER(?)", &username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
