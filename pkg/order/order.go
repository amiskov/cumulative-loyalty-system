package order

import "time"

type Order struct {
	Number     string    `json:"number"`
	UserId     string    `json:"-"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}
