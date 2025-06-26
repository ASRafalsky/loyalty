package models

import (
	"time"
)

//go:generate easyjson --all entry.go
type Order struct {
	ID     string    `json:"number"`
	Status string    `json:"status,omitempty"`
	Points *float64  `json:"accrual,omitempty"`
	TS     time.Time `json:"uploaded_at"`
}

type Accrual struct {
	ID     string   `json:"order"`
	Status string   `json:"status,omitempty"`
	Points *float64 `json:"accrual,omitempty"`
}

type InputWithdraw struct {
	ID     string  `json:"order"`
	Points float64 `json:"sum"`
}

type Withdraw struct {
	ID     string    `json:"order"`
	Points float64   `json:"sum"`
	TS     time.Time `json:"processed_at"`
}

type UserStats struct {
	CurrentPoints  *float64 `json:"current,omitempty"`
	WithdrawPoints *float64 `json:"withdrawn,omitempty"`
}

type Credential struct {
	Login    string `validate:"required,min=3,max=50" json:"login"`
	Password string `validate:"required,min=4,max=72" json:"password"`
}
