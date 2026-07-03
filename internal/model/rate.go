package model

import "time"

type Rate struct {
	Price     float64
	Timestamp time.Time
	Source    string
}
