package media_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"erdinhrmwn/bangunin/pkg/apperr"

	svcmocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	mediausecase "erdinhrmwn/bangunin/internal/usecase/media"
)

func newUsecase(t *testing.T) (*mediausecase.Usecase, *svcmocks.MockStorageService, *svcmocks.MockMediaEnqueuer) {
	t.Helper()
	storage := svcmocks.NewMockStorageService(t)
	enqueuer := svcmocks.NewMockMediaEnqueuer(t)
	return mediausecase.New(storage, enqueuer), storage, enqueuer
}

func TestUpload_InvalidScope(t *testing.T) {
	uc, _, _ := newUsecase(t)
	_, err := uc.Upload(context.Background(), "../../etc", "image/png", 100, bytes.NewReader(nil))
	assert.Equal(t, "INVALID_SCOPE", apperr.From(err).Code)
}

func TestUpload_UnsupportedType(t *testing.T) {
	uc, _, _ := newUsecase(t)
	_, err := uc.Upload(context.Background(), "kyc", "application/x-msdownload", 100, bytes.NewReader(nil))
	assert.Equal(t, "UNSUPPORTED_MEDIA_TYPE", apperr.From(err).Code)
}

func TestUpload_ImageTooLarge(t *testing.T) {
	uc, _, _ := newUsecase(t)
	_, err := uc.Upload(context.Background(), "kyc", "image/png", 6<<20, bytes.NewReader(nil))
	assert.Equal(t, "FILE_TOO_LARGE", apperr.From(err).Code)
}

func TestUpload_PDFTooLarge(t *testing.T) {
	uc, _, _ := newUsecase(t)
	_, err := uc.Upload(context.Background(), "kyc", "application/pdf", 20<<20, bytes.NewReader(nil))
	assert.Equal(t, "FILE_TOO_LARGE", apperr.From(err).Code)
}

func TestUpload_ImageEnqueuesResize(t *testing.T) {
	uc, storage, enqueuer := newUsecase(t)
	storage.EXPECT().Upload(mock.Anything, mock.Anything, mock.Anything, int64(100), "image/jpeg").
		Return("https://cdn/kyc/x.jpg", nil)
	enqueuer.EXPECT().EnqueueMediaProcess(mock.Anything).Return(nil)

	res, err := uc.Upload(context.Background(), "kyc", "image/jpeg", 100, bytes.NewReader(nil))

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/kyc/x.jpg", res.URL)
}

func TestUpload_PDFSkipsResize(t *testing.T) {
	uc, storage, _ := newUsecase(t)
	storage.EXPECT().Upload(mock.Anything, mock.Anything, mock.Anything, int64(100), "application/pdf").
		Return("https://cdn/kyc/x.pdf", nil)

	res, err := uc.Upload(context.Background(), "kyc", "application/pdf", 100, bytes.NewReader(nil))

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/kyc/x.pdf", res.URL)
}
