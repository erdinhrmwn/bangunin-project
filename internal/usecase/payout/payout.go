// Package payout implements supplier withdraw requests, admin approval with
// synchronous disbursement, and the settlement ledger reader (FR-6.3/FR-6.4).
package payout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/internal/pkg/notify"
	"erdinhrmwn/bangunin/pkg/apperr"
)

// minWithdrawAmount: ponytail: fixed value, promote to config if ops needs
// per-environment tuning (CLAUDE.md §5.9 rung).
const minWithdrawAmount = 50_000

type Usecase struct {
	withdraws repository.WithdrawRequestRepository
	balances  repository.SupplierBalanceRepository
	banks     repository.SupplierBankAccountRepository
	suppliers repository.SupplierRepository
	users     repository.UserRepository
	audit     repository.AuditLogRepository
	notify    repository.NotificationRepository
	email     service.EmailEnqueuer
	ledger    repository.LedgerEntryRepository
	payment   service.PaymentGateway
}

func New(
	withdraws repository.WithdrawRequestRepository,
	balances repository.SupplierBalanceRepository,
	banks repository.SupplierBankAccountRepository,
	suppliers repository.SupplierRepository,
	users repository.UserRepository,
	audit repository.AuditLogRepository,
	notify repository.NotificationRepository,
	email service.EmailEnqueuer,
	ledger repository.LedgerEntryRepository,
	payment service.PaymentGateway,
) *Usecase {
	return &Usecase{
		withdraws: withdraws, balances: balances, banks: banks, suppliers: suppliers,
		users: users, audit: audit, notify: notify, email: email, ledger: ledger, payment: payment,
	}
}

func (u *Usecase) Balance(ctx context.Context, supplierID uuid.UUID) (*entity.SupplierBalance, error) {
	b, err := u.balances.FindBySupplierID(ctx, supplierID)
	if errors.Is(err, errs.ErrNotFound) {
		return &entity.SupplierBalance{SupplierID: supplierID}, nil
	}
	return b, err
}

func (u *Usecase) ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.WithdrawRequest, int, error) {
	return u.withdraws.ListBySupplier(ctx, supplierID, page, perPage)
}

func (u *Usecase) ListAll(ctx context.Context, status string, page, perPage int) ([]*entity.WithdrawRequest, int, error) {
	return u.withdraws.ListAll(ctx, status, page, perPage)
}

func (u *Usecase) Get(ctx context.Context, id uuid.UUID) (*entity.WithdrawRequest, error) {
	return u.withdraws.FindByID(ctx, id)
}

// GetForSupplier enforces ownership (FR-6.3), mirroring order usecase's per-actor checks.
func (u *Usecase) GetForSupplier(ctx context.Context, id, supplierID uuid.UUID) (*entity.WithdrawRequest, error) {
	w, err := u.withdraws.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.SupplierID != supplierID {
		return nil, apperr.New("FORBIDDEN", "Withdraw request does not belong to your store", 403)
	}
	return w, nil
}

// Request validates amount + bank ownership and holds the funds immediately
// by decrementing the balance (FR-6.3). No separate "pending_withdraw"
// ledger row — the hold lives on supplier_balances + the request's status.
func (u *Usecase) Request(ctx context.Context, supplierID, bankAccountID uuid.UUID, amount float64) (*entity.WithdrawRequest, error) {
	if amount < minWithdrawAmount {
		return nil, apperr.New("VALIDATION_ERROR", fmt.Sprintf("Minimum withdrawal amount is Rp%.0f", float64(minWithdrawAmount)), 422)
	}

	bank, err := u.banks.FindByID(ctx, bankAccountID)
	if err != nil {
		return nil, err
	}
	if bank.SupplierID != supplierID {
		return nil, apperr.New("FORBIDDEN", "Bank account does not belong to your store", 403)
	}

	balance, err := u.Balance(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if amount > balance.Balance {
		return nil, apperr.New("VALIDATION_ERROR", "Insufficient balance", 422)
	}

	if _, err := u.balances.Increment(ctx, supplierID, -amount); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("payout: generate withdraw id: %w", err)
	}
	w := &entity.WithdrawRequest{
		ID: id, SupplierID: supplierID, BankAccountID: bankAccountID, Amount: amount,
		Status: entity.WithdrawStatusPending, RequestedAt: time.Now(),
	}
	if err := u.withdraws.Create(ctx, w); err != nil {
		_, _ = u.balances.Increment(ctx, supplierID, amount) // best-effort rollback of the hold
		return nil, err
	}
	return w, nil
}

// Cancel lets the supplier withdraw a still-pending request, releasing the hold (FR-6.3).
func (u *Usecase) Cancel(ctx context.Context, id, supplierID uuid.UUID) error {
	w, err := u.GetForSupplier(ctx, id, supplierID)
	if err != nil {
		return err
	}
	if w.Status != entity.WithdrawStatusPending {
		return apperr.New("INVALID_STATUS", "Withdraw request is not pending", 409)
	}
	if _, err := u.balances.Increment(ctx, supplierID, w.Amount); err != nil {
		return err
	}
	w.Status = entity.WithdrawStatusRejected
	now := time.Now()
	w.ProcessedAt = &now
	return u.withdraws.Update(ctx, w)
}

// Approve calls the (mock-mode, synchronous) payment-service disbursement
// inline and settles the result immediately.
//
// ponytail: PaymentGateway.Disburse is synchronous in every current
// implementation (mock Xendit client always succeeds inline). Modeling this
// as sync — rather than scaffolding a "processing" status + a separate
// /internal/withdraws/callback endpoint — avoids an unused async path.
// Upgrade when a real Xendit disbursement integration is genuinely async:
// set status "processing" here, add the callback handler, settle there.
func (u *Usecase) Approve(ctx context.Context, actorID, id uuid.UUID, ip string) (*entity.WithdrawRequest, error) {
	w, err := u.withdraws.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.Status != entity.WithdrawStatusPending {
		return nil, apperr.New("INVALID_STATUS", "Withdraw request is not pending", 409)
	}
	bank, err := u.banks.FindByID(ctx, w.BankAccountID)
	if err != nil {
		return nil, err
	}

	before := w.Status
	now := time.Now()
	disbID, disbErr := u.payment.Disburse(ctx, w.ID, w.Amount, bank.BankCode, bank.AccountNumber, bank.AccountName)
	if disbErr != nil {
		if _, err := u.balances.Increment(ctx, w.SupplierID, w.Amount); err != nil {
			return nil, err
		}
		w.Status = entity.WithdrawStatusFailed
		w.ProcessedAt = &now
		if err := u.withdraws.Update(ctx, w); err != nil {
			return nil, err
		}
		if err := u.recordAndNotify(ctx, actorID, w, "withdraw.failed", ip, disbErr.Error(), before); err != nil {
			return nil, err
		}
		return w, nil
	}

	w.Status = entity.WithdrawStatusDisbursed
	w.DisbursementID = disbID
	w.ProcessedAt = &now
	if err := u.withdraws.Update(ctx, w); err != nil {
		return nil, err
	}

	balance, err := u.Balance(ctx, w.SupplierID)
	if err != nil {
		return nil, err
	}
	if err := u.ledger.Create(ctx, &entity.LedgerEntry{
		ID: mustNewV7(), WithdrawRequestID: &w.ID, SupplierID: w.SupplierID,
		Type: entity.LedgerTypeDebitWithdraw, Amount: w.Amount, BalanceAfter: balance.Balance,
	}); err != nil {
		return nil, err
	}

	if err := u.recordAndNotify(ctx, actorID, w, "withdraw.approve", ip, "", before); err != nil {
		return nil, err
	}
	return w, nil
}

// Reject restores the hold and never calls payment-service (FR-6.4).
func (u *Usecase) Reject(ctx context.Context, actorID, id uuid.UUID, reason, ip string) (*entity.WithdrawRequest, error) {
	w, err := u.withdraws.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.Status != entity.WithdrawStatusPending {
		return nil, apperr.New("INVALID_STATUS", "Withdraw request is not pending", 409)
	}

	before := w.Status
	if _, err := u.balances.Increment(ctx, w.SupplierID, w.Amount); err != nil {
		return nil, err
	}
	now := time.Now()
	w.Status = entity.WithdrawStatusRejected
	w.ProcessedAt = &now
	if err := u.withdraws.Update(ctx, w); err != nil {
		return nil, err
	}
	if err := u.recordAndNotify(ctx, actorID, w, "withdraw.reject", ip, reason, before); err != nil {
		return nil, err
	}
	return w, nil
}

func (u *Usecase) recordAndNotify(ctx context.Context, actorID uuid.UUID, w *entity.WithdrawRequest, action, ip, reason, before string) error {
	auditID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("payout: generate audit id: %w", err)
	}
	if err := u.audit.Create(ctx, &entity.AuditLog{
		ID: auditID, ActorID: actorID, Action: action, EntityType: "withdraw_request", EntityID: w.ID,
		Metadata:  map[string]any{"reason": reason, "before": before, "after": w.Status, "amount": w.Amount},
		IPAddress: ip,
	}); err != nil {
		return err
	}

	s, err := u.suppliers.FindByID(ctx, w.SupplierID)
	if err != nil {
		return err
	}

	title, body := withdrawNotificationText(action, reason, w.Amount)
	return notify.SendWithEmail(ctx, u.notify, u.users, u.email, s.UserID, entity.NotificationTypePayout, title, body)
}

func withdrawNotificationText(action, reason string, amount float64) (title, body string) {
	switch action {
	case "withdraw.approve":
		return "Withdrawal disbursed", fmt.Sprintf("Your withdrawal of Rp%.0f has been disbursed.", amount)
	case "withdraw.failed":
		return "Withdrawal failed", fmt.Sprintf("Your withdrawal of Rp%.0f failed and the amount has been returned to your balance: %s", amount, reason)
	default:
		return "Withdrawal rejected", fmt.Sprintf("Your withdrawal of Rp%.0f was rejected: %s", amount, reason)
	}
}

func mustNewV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}
