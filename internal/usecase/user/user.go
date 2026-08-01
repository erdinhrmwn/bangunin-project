// Package user implements basic profile operations (FR-2.9).
package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/hash"
)

type Usecase struct {
	users     repository.UserRepository
	addresses repository.UserAddressRepository
}

func New(users repository.UserRepository, addresses repository.UserAddressRepository) *Usecase {
	return &Usecase{users: users, addresses: addresses}
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

func (u *Usecase) UpdateNotificationSettings(ctx context.Context, id uuid.UUID, emailMarketing bool) (*entity.User, error) {
	usr, err := u.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	usr.EmailMarketing = emailMarketing
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

// AddressInput is the address create/update payload (FR-5.1 address book).
type AddressInput struct {
	Label          string
	RecipientName  string
	RecipientPhone string
	ProvinceID     int
	CityID         int
	Subdistrict    string
	PostalCode     string
	AddressDetail  string
	IsDefault      bool
}

func (u *Usecase) ListAddresses(ctx context.Context, userID uuid.UUID) ([]*entity.UserAddress, error) {
	return u.addresses.ListByUserID(ctx, userID)
}

func (u *Usecase) CreateAddress(ctx context.Context, userID uuid.UUID, in AddressInput) (*entity.UserAddress, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("user: generate address id: %w", err)
	}
	a := &entity.UserAddress{
		ID:             id,
		UserID:         userID,
		Label:          in.Label,
		RecipientName:  in.RecipientName,
		RecipientPhone: in.RecipientPhone,
		ProvinceID:     in.ProvinceID,
		CityID:         in.CityID,
		Subdistrict:    in.Subdistrict,
		PostalCode:     in.PostalCode,
		AddressDetail:  in.AddressDetail,
		IsDefault:      in.IsDefault,
	}
	if err := u.addresses.Create(ctx, a); err != nil {
		return nil, err
	}
	if in.IsDefault {
		if err := u.addresses.ClearDefault(ctx, userID, a.ID); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (u *Usecase) UpdateAddress(ctx context.Context, userID, id uuid.UUID, in AddressInput) (*entity.UserAddress, error) {
	a, err := u.ownedAddress(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	a.Label = in.Label
	a.RecipientName = in.RecipientName
	a.RecipientPhone = in.RecipientPhone
	a.ProvinceID = in.ProvinceID
	a.CityID = in.CityID
	a.Subdistrict = in.Subdistrict
	a.PostalCode = in.PostalCode
	a.AddressDetail = in.AddressDetail
	a.IsDefault = in.IsDefault
	if err := u.addresses.Update(ctx, a); err != nil {
		return nil, err
	}
	if in.IsDefault {
		if err := u.addresses.ClearDefault(ctx, userID, a.ID); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (u *Usecase) DeleteAddress(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := u.ownedAddress(ctx, userID, id); err != nil {
		return err
	}
	return u.addresses.Delete(ctx, id)
}

func (u *Usecase) ownedAddress(ctx context.Context, userID, id uuid.UUID) (*entity.UserAddress, error) {
	a, err := u.addresses.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, apperr.New("FORBIDDEN", "Address does not belong to you", 403)
	}
	return a, nil
}
