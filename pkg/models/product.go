package models

import (
    "github.com/google/uuid"
    "time"
)

type Product struct {
    ID           uuid.UUID
    Name         string
    Price        float64
    Description  string
    Image        string
    CreatedDate  time.Time
    ModifiedDate time.Time
}
