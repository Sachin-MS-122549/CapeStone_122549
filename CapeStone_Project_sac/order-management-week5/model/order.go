package model

import "time"

type OrderStatus string

const (
	StatusCreated   OrderStatus = "CREATED"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID          int64       `json:"id" db:"id"`
	CustomerID  int64       `json:"customerId" db:"customer_id"`
	ProductID   int64       `json:"productId" db:"product_id"`
	Quantity    int         `json:"quantity" db:"quantity"`
	UnitPrice   float64     `json:"unitPrice" db:"unit_price"`
	TotalAmount float64     `json:"totalAmount" db:"total_amount"`
	Status      OrderStatus `json:"status" db:"status"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time   `json:"updatedAt" db:"updated_at"`
}
type CreateOrderRequest struct {
	CustomerID int64 `json:"customerId"`
	ProductID  int64 `json:"productId"`
	Quantity   int   `json:"quantity"`
}
type UpdateStatusRequest struct {
	Status OrderStatus `json:"status"`
}
