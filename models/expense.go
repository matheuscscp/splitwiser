package models

type (
	// Expense ...
	Expense struct {
		Cost        PriceInCents
		UserShares  [2]*UserShare
		Description string
	}

	// UserShare ...
	UserShare struct {
		User ReceiptItemOwner
		Paid PriceInCents
		Owed PriceInCents
	}
)
