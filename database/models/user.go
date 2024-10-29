package models

type User struct {
	SenderName   string `json:"sender" validate:"required,min=3,max=40"`
	ReceiverName string `json:"receiver" validate:"required,min=3,max=40"`
	Address      string `json:"address" validate:"required,min=3,max=100"`
	Date         string `json:"date" validate:"required"`
	Content      string `json:"content" validate:"required,min=3,max=100"`
	AccountID    uint   `json:"account_id"`
}
