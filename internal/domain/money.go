package domain

import "errors"

type Money struct {
	Amount   int64
	Currency string
}

var ErrInvalidMoney = errors.New("invalid money")

func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 || currency == "" {
		return Money{}, ErrInvalidMoney
	}

	return Money{Amount: amount, Currency: currency}, nil
}
