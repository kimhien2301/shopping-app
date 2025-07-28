package models

import (
    "github.com/google/uuid"
    "time"
)

type Order struct {
    ID     uuid.UUID
    UserID string
    Status OrderStatus
    Date   time.Time
    Items  []OrderItem
    Total  float64
}

type OrderItem struct {
    OrderID   uuid.UUID
    ProductID uuid.UUID
    Quantity  int
    Product   *Product
    Cost      float64
}

type OrderStatus string

const (
    Ordered   OrderStatus = "ordered"
    Pending   OrderStatus = "pending"
    Shipped   OrderStatus = "shipped"
    Delivered OrderStatus = "delivered"
    Cancel    OrderStatus = "cancel"
)
