package order

import "time"

type Order struct {
	Number     string    `json:"number"`
	UserID     string    `json:"-"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

const (
	PROCESSED  = "PROCESSED"
	NEW        = "NEW"
	INVALID    = "INVALID"
	PROCESSING = "PROCESSING"
)
