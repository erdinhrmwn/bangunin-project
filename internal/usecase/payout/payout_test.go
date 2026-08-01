package payout_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	servicemocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	payoutusecase "erdinhrmwn/bangunin/internal/usecase/payout"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type deps struct {
	withdraws *mocks.MockWithdrawRequestRepository
	balances  *mocks.MockSupplierBalanceRepository
	banks     *mocks.MockSupplierBankAccountRepository
	suppliers *mocks.MockSupplierRepository
	users     *mocks.MockUserRepository
	audit     *mocks.MockAuditLogRepository
	notify    *mocks.MockNotificationRepository
	email     *servicemocks.MockEmailEnqueuer
	ledger    *mocks.MockLedgerEntryRepository
	payment   *servicemocks.MockPaymentGateway
}

func newUsecase(t *testing.T) (*payoutusecase.Usecase, *deps) {
	t.Helper()
	d := &deps{
		withdraws: mocks.NewMockWithdrawRequestRepository(t),
		balances:  mocks.NewMockSupplierBalanceRepository(t),
		banks:     mocks.NewMockSupplierBankAccountRepository(t),
		suppliers: mocks.NewMockSupplierRepository(t),
		users:     mocks.NewMockUserRepository(t),
		audit:     mocks.NewMockAuditLogRepository(t),
		notify:    mocks.NewMockNotificationRepository(t),
		email:     servicemocks.NewMockEmailEnqueuer(t),
		ledger:    mocks.NewMockLedgerEntryRepository(t),
		payment:   servicemocks.NewMockPaymentGateway(t),
	}
	uc := payoutusecase.New(
		d.withdraws, d.balances, d.banks, d.suppliers, d.users,
		d.audit, d.notify, d.email, d.ledger, d.payment,
	)
	return uc, d
}

// stubNotify allows recordAndNotify's audit+notification+email calls for any withdraw.
func stubNotify(d *deps) {
	d.audit.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	d.suppliers.EXPECT().FindByID(mock.Anything, mock.Anything).Return(&entity.Supplier{UserID: uuid.Must(uuid.NewV7())}, nil).Maybe()
	d.notify.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	d.users.EXPECT().FindByID(mock.Anything, mock.Anything).Return(&entity.User{Email: "supplier@example.com"}, nil).Maybe()
	d.email.EXPECT().EnqueueEmail(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestRequest_BelowMinimum_Returns422(t *testing.T) {
	uc, _ := newUsecase(t)
	supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	_, err := uc.Request(context.Background(), supplierID, bankID, 10_000)

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestRequest_BankAccountNotOwned_Returns403(t *testing.T) {
	uc, d := newUsecase(t)
	supplierID, otherSupplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID, SupplierID: otherSupplierID}, nil)

	_, err := uc.Request(context.Background(), supplierID, bankID, 100_000)

	require.Equal(t, "FORBIDDEN", appErrCode(t, err))
}

func TestRequest_InsufficientBalance_Returns422(t *testing.T) {
	uc, d := newUsecase(t)
	supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID, SupplierID: supplierID}, nil)
	d.balances.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return(&entity.SupplierBalance{SupplierID: supplierID, Balance: 50_000}, nil)

	_, err := uc.Request(context.Background(), supplierID, bankID, 100_000)

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestRequest_Success_HoldsBalance(t *testing.T) {
	uc, d := newUsecase(t)
	supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID, SupplierID: supplierID}, nil)
	d.balances.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return(&entity.SupplierBalance{SupplierID: supplierID, Balance: 500_000}, nil)
	d.balances.EXPECT().Increment(mock.Anything, supplierID, float64(-100_000)).Return(400_000, nil)
	d.withdraws.EXPECT().Create(mock.Anything, mock.MatchedBy(func(w *entity.WithdrawRequest) bool {
		return w.SupplierID == supplierID && w.BankAccountID == bankID && w.Amount == 100_000 && w.Status == entity.WithdrawStatusPending
	})).Return(nil)

	w, err := uc.Request(context.Background(), supplierID, bankID, 100_000)

	require.NoError(t, err)
	require.Equal(t, entity.WithdrawStatusPending, w.Status)
}

func TestRequest_NoBalanceRow_TreatsAsZero(t *testing.T) {
	uc, d := newUsecase(t)
	supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID, SupplierID: supplierID}, nil)
	d.balances.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return(nil, errs.ErrNotFound)

	_, err := uc.Request(context.Background(), supplierID, bankID, 100_000)

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestCancel_Pending_ReleasesHold(t *testing.T) {
	uc, d := newUsecase(t)
	id, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(&entity.WithdrawRequest{ID: id, SupplierID: supplierID, Amount: 100_000, Status: entity.WithdrawStatusPending}, nil)
	d.balances.EXPECT().Increment(mock.Anything, supplierID, float64(100_000)).Return(600_000, nil)
	d.withdraws.EXPECT().Update(mock.Anything, mock.MatchedBy(func(w *entity.WithdrawRequest) bool {
		return w.Status == entity.WithdrawStatusRejected && w.ProcessedAt != nil
	})).Return(nil)

	err := uc.Cancel(context.Background(), id, supplierID)

	require.NoError(t, err)
}

func TestCancel_NotPending_Returns409(t *testing.T) {
	uc, d := newUsecase(t)
	id, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(&entity.WithdrawRequest{ID: id, SupplierID: supplierID, Status: entity.WithdrawStatusDisbursed}, nil)

	err := uc.Cancel(context.Background(), id, supplierID)

	require.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}

func TestApprove_Success_DisbursesAndRecordsLedger(t *testing.T) {
	uc, d := newUsecase(t)
	stubNotify(d)
	adminID, id, supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	w := &entity.WithdrawRequest{ID: id, SupplierID: supplierID, BankAccountID: bankID, Amount: 100_000, Status: entity.WithdrawStatusPending}
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(w, nil)
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID, BankCode: "bca", AccountNumber: "123", AccountName: "Toko A"}, nil)
	d.payment.EXPECT().Disburse(mock.Anything, id, 100_000.0, "bca", "123", "Toko A").Return("disb-1", nil)
	d.withdraws.EXPECT().Update(mock.Anything, mock.MatchedBy(func(w *entity.WithdrawRequest) bool {
		return w.Status == entity.WithdrawStatusDisbursed && w.DisbursementID == "disb-1" && w.ProcessedAt != nil
	})).Return(nil)
	d.balances.EXPECT().FindBySupplierID(mock.Anything, supplierID).Return(&entity.SupplierBalance{SupplierID: supplierID, Balance: 400_000}, nil)
	d.ledger.EXPECT().Create(mock.Anything, mock.MatchedBy(func(e *entity.LedgerEntry) bool {
		return e.WithdrawRequestID != nil && *e.WithdrawRequestID == id && e.Type == entity.LedgerTypeDebitWithdraw && e.Amount == 100_000 && e.BalanceAfter == 400_000
	})).Return(nil)

	got, err := uc.Approve(context.Background(), adminID, id, "127.0.0.1")

	require.NoError(t, err)
	require.Equal(t, entity.WithdrawStatusDisbursed, got.Status)
}

func TestApprove_DisburseFails_RestoresBalanceAndMarksFailed(t *testing.T) {
	uc, d := newUsecase(t)
	stubNotify(d)
	adminID, id, supplierID, bankID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	w := &entity.WithdrawRequest{ID: id, SupplierID: supplierID, BankAccountID: bankID, Amount: 100_000, Status: entity.WithdrawStatusPending}
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(w, nil)
	d.banks.EXPECT().FindByID(mock.Anything, bankID).Return(&entity.SupplierBankAccount{ID: bankID}, nil)
	d.payment.EXPECT().Disburse(mock.Anything, id, 100_000.0, "", "", "").Return("", errors.New("insufficient gateway balance"))
	d.balances.EXPECT().Increment(mock.Anything, supplierID, float64(100_000)).Return(500_000, nil)
	d.withdraws.EXPECT().Update(mock.Anything, mock.MatchedBy(func(w *entity.WithdrawRequest) bool {
		return w.Status == entity.WithdrawStatusFailed && w.ProcessedAt != nil
	})).Return(nil)

	got, err := uc.Approve(context.Background(), adminID, id, "127.0.0.1")

	require.NoError(t, err)
	require.Equal(t, entity.WithdrawStatusFailed, got.Status)
}

func TestApprove_NotPending_Returns409(t *testing.T) {
	uc, d := newUsecase(t)
	adminID, id := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(&entity.WithdrawRequest{ID: id, Status: entity.WithdrawStatusRejected}, nil)

	_, err := uc.Approve(context.Background(), adminID, id, "127.0.0.1")

	require.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}

func TestReject_Pending_RestoresBalanceNoPaymentCall(t *testing.T) {
	uc, d := newUsecase(t)
	stubNotify(d)
	adminID, id, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(&entity.WithdrawRequest{ID: id, SupplierID: supplierID, Amount: 100_000, Status: entity.WithdrawStatusPending}, nil)
	d.balances.EXPECT().Increment(mock.Anything, supplierID, float64(100_000)).Return(500_000, nil)
	d.withdraws.EXPECT().Update(mock.Anything, mock.MatchedBy(func(w *entity.WithdrawRequest) bool {
		return w.Status == entity.WithdrawStatusRejected && w.ProcessedAt != nil
	})).Return(nil)

	got, err := uc.Reject(context.Background(), adminID, id, "duplicate request", "127.0.0.1")

	require.NoError(t, err)
	require.Equal(t, entity.WithdrawStatusRejected, got.Status)
	d.payment.AssertNotCalled(t, "Disburse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReject_NotPending_Returns409(t *testing.T) {
	uc, d := newUsecase(t)
	adminID, id := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.withdraws.EXPECT().FindByID(mock.Anything, id).Return(&entity.WithdrawRequest{ID: id, Status: entity.WithdrawStatusFailed}, nil)

	_, err := uc.Reject(context.Background(), adminID, id, "reason", "127.0.0.1")

	require.Equal(t, "INVALID_STATUS", appErrCode(t, err))
}
