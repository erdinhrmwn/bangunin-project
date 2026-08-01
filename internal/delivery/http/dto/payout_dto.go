package dto

type RequestWithdrawRequest struct {
	BankAccountID string  `json:"bank_account_id" validate:"required,uuid"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
}
