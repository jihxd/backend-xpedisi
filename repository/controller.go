package repository

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jihxd/ekspedisi/database/migrations"
	"github.com/jihxd/ekspedisi/database/models"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/go-playground/validator.v9"
	"gorm.io/gorm/clause"
)

var jwtSecret = "sk"

type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

var validate = validator.New()

func ValidateStruct(user models.User) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(user)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}

	return errors
}

func (repo *Repository) Register(c *fiber.Ctx) error {
	type RegisterInput struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required,min=6"`
		Email    string `json:"email" validate:"required,email"`
	}

	var input RegisterInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Validasi email
	if !govalidator.IsEmail(input.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format Email Tidak Valid"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	account := migrations.Account{
		Username: input.Username,
		Password: string(hashedPassword),
		Email:    input.Email,
	}

	if err := repo.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user"})
	}

	// Buat JWT token untuk account yang baru dibuat
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountID": account.ID,
		"exp":       time.Now().Add(time.Hour * 72).Unix(),
	})

	// Sign token dengan secret
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}

	// nyimoen token di Redis
	accountIDStr := strconv.Itoa(int(account.ID))
	redisKey := "token:" + accountIDStr

	err = repo.RedisClient.Set(c.Context(), redisKey, tokenString, 24*time.Hour).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store session in Redis"})
	}

	// Log sjmpen token di Redis
	fmt.Println("Token baru disimpan di Redis untuk account ID:", accountIDStr)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"token":   tokenString,
	})
}

// Login Handler

func (repo *Repository) Login(c *fiber.Ctx) error {
	type LoginInput struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Ambil user dari database berdasarkan username
	var account migrations.Account
	if err := repo.DB.Where("username = ?", input.Username).First(&account).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Username or password incorrect"})
	}

	// Bandingkan password yang di-hash
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Username or password incorrect"})
	}

	// Konversi account.ID ke string untuk Redis key
	accountIDStr := strconv.Itoa(int(account.ID))
	redisKey := "token:" + accountIDStr

	// Cek apakah token sudah ada di Redis
	cachedToken, err := repo.RedisClient.Get(c.Context(), redisKey).Result()
	if err == nil && cachedToken != "" {
		// Jika token ditemukan di Redis, log ke terminal dan kembalikan token
		fmt.Println("Token diambil dari Redis untuk account ID:", accountIDStr)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Login successful (from Redis)",
			"token":   cachedToken,
		})
	}

	// Jika token tidak ditemukan di Redis, buat token baru
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountID": account.ID,
		"exp":       time.Now().Add(time.Hour * 72).Unix(),
	})

	// Sign token dengan secret
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}

	// Simpan token di Redis
	err = repo.RedisClient.Set(c.Context(), redisKey, tokenString, 24*time.Hour).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store session"})
	}

	// Log bahwa token disimpan di Redis
	fmt.Println("Token baru disimpan di Redis untuk account ID:", accountIDStr)

	// Kembalikan respons sukses beserta tokennya
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"token":   tokenString,
	})
}

func JWTMiddleware(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid token"})
	}

	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	accountID, ok := claims["accountID"].(float64)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "AccountID not found in token"})
	}

	c.Locals("accountID", uint(accountID))
	return c.Next()
}

func (r *Repository) GetPaket(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")

	// Parsing token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	accountID := claims["accountID"].(float64)
	id := uint(accountID)

	// Query paket berdasarkan AccountID yang login
	var users []migrations.Users
	err = r.DB.Where("account_id = ?", id).Find(&users).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data paket",
		})
	}

	return c.JSON(users)
}

func (r *Repository) CreatePaket(c *fiber.Ctx) error {
	// Ambil AccountID dari sesi atau token
	accountID, ok := c.Locals("accountID").(uint)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized: accountID not found",
		})
	}

	user := models.User{}
	err := c.BodyParser(&user)
	if err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
			"message": "Request failed",
		})
	}

	// Set AccountID ke data user
	user.AccountID = accountID

	if err := r.DB.Create(&user).Error; err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menambah paket",
			"data":    err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Paket ditambahkan",
		"data":    user,
	})
}

func (r *Repository) MarkPaketAsDone(context *fiber.Ctx) error {
	id := context.Params("id")
	if id == "" {
		return context.Status(fiber.StatusBadRequest).JSON(&fiber.Map{"message": "ID tidak boleh kosong"})
	}

	// Ambil data paket berdasarkan ID
	var user migrations.Users
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "Paket tidak ditemukan"})
	}

	currentTime := time.Now()
	if err := r.DB.Model(&user).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       "selesai",
		"arrival_date": currentTime,
	}).Error; err != nil {
		return context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "Gagal mengubah status"})
	}

	return context.Status(http.StatusOK).JSON(&fiber.Map{
		"message":      "Status berhasil diubah menjadi selesai",
		"arrival_date": currentTime,
	})
}

func (r *Repository) UpdatePaket(context *fiber.Ctx) error {
	user := models.User{}
	err := context.BodyParser(&user)
	if err != nil {
		context.Status(http.StatusUnprocessableEntity).JSON(
			&fiber.Map{"message": "Request Failed"})

		return err
	}
	errors := ValidateStruct(user)
	if errors != nil {
		return context.Status(fiber.StatusBadRequest).JSON(errors)
	}

	db := r.DB
	id := context.Params("id")

	if id == "" {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "ID tidak boleh kosong"})
		return nil
	}

	if db.Model(&user).Where("id = ?", id).Updates(&user).RowsAffected == 0 {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "tidak bisa melacak paket"})
		return nil
	}

	context.Status(http.StatusOK).JSON(&fiber.Map{"status": "success", "message": "paket berhasil di update"})
	return nil
}

func (r *Repository) DeletePaket(context *fiber.Ctx) error {
	userModel := migrations.Users{}
	id := context.Params("id")

	if id == "" {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "ID tidak boleh kosong"})
		return nil
	}

	err := r.DB.Delete(userModel, id)

	if err.Error != nil {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "tidak bisa menghapus"})
		return err.Error
	}

	context.Status(http.StatusOK).JSON(&fiber.Map{"status": "success", "message": "paket telah sampai"})
	return nil
}

func (r *Repository) GetPaketByID(context *fiber.Ctx) error {
	userModel := &migrations.Users{}
	id := context.Params("id")

	if id == "" {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "ID tidak boleh kosong"})
		return nil
	}

	err := r.DB.Where("id = ?", id).First(&userModel).Error

	if err != nil {
		context.Status(http.StatusBadRequest).JSON(&fiber.Map{"message": "tidak bisa mencari paket"})
		return err
	}

	context.Status(http.StatusOK).JSON(&fiber.Map{"status": "success", "message": "paket ditemukan", "data": userModel})
	return nil

}
