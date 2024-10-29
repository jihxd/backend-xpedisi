package migrations

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Users struct {
	ID           uint       `gorm:"primarykey;autoIncrement" json:"id"`
	SenderName   string     `json:"sender"`
	ReceiverName string     `json:"receiver"`
	Address      string     `json:"address"`
	Date         time.Time  `json:"date"`
	ArrivalDate  *time.Time `json:"arrival"`
	Content      string     `json:"content"`
	Status       string     `json:"status" gorm:"default:'dikirim'"`
	AccountID    uint       `json:"account_id"`
	Account      Account    `gorm:"foreignKey:AccountID"`
}
type Account struct {
	ID       uint    `gorm:"primarykey;autoIncrement" json:"id"`
	Username string  `json:"username" gorm:"unique"`
	Password string  `json:"password"`
	Email    string  `json:"email" gorm:"unique"`
	Users    []Users `gorm:"foreignKey:AccountID"`
}

// buat hashh
func (a *Account) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.Password = string(hashedPassword)
	return nil
}

func (a *Account) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(providedPassword))
	return err
}

func MigrateUsersAndAccounts(db *gorm.DB) error {
	err := db.AutoMigrate(&Users{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&Account{})
	return err
}
