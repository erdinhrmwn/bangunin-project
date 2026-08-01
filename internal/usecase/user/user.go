// Package user implements basic profile operations (FR-2.9).
package user

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
)

type Usecase struct {
	users repository.UserRepository
}

func New(users repository.UserRepository) *Usecase {
	return &Usecase{users: users}
}

func (u *Usecase) Me(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return u.users.FindByID(ctx, id)
}

func (u *Usecase) UpdateProfile(ctx context.Context, id uuid.UUID, name, phone string) (*entity.User, error) {
	usr, err := u.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	usr.Name = name
	usr.Phone = phone
	if err := u.users.Update(ctx, usr); err != nil {
		return nil, err
	}
	return usr, nil
}

func (u *Usecase) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error {
	usr, err := u.users.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !hash.Compare(usr.PasswordHash, oldPassword) {
		return apperr.New("INVALID_PASSWORD", "Old password is incorrect", 422)
	}

	hashed, err := hash.Hash(newPassword)
	if err != nil {
		return err
	}
	usr.PasswordHash = hashed
	return u.users.Update(ctx, usr)
}
