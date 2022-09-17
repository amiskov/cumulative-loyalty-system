package balance

import "time"

type Withdraw struct {
	Order       string    `json:"order"`
	UserID      string    `json:"-"`
	Sum         float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}
