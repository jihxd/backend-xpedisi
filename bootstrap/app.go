package bootstrap

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jihxd/ekspedisi/database/migrations"
	"github.com/jihxd/ekspedisi/database/storage"
	"github.com/jihxd/ekspedisi/repository"
	"github.com/joho/godotenv"
)

var RedisClient *redis.Client

func InitializeApp(app *fiber.App) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	// Inisialisasi Redis client
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Sesuaikan alamat Redis jika berbeda
		Password: "",               // Password jika ada, kosongkan jika tidak ada
		DB:       0,                // Database Redis yang digunakan
	})

	// Cek koneksi Redis
	_, err = RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("couldn't connect to Redis:", err)
	}

	// Konfigurasi koneksi database
	config := &storage.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	db, err := storage.NewConnection(config)
	if err != nil {
		log.Fatal("couldn't load database")
	}

	// Migrate database
	err = migrations.MigrateUsersAndAccounts(db)
	if err != nil {
		log.Fatal("couldn't load migrate db")
	}

	// Inisialisasi repo dan setup route
	repo := repository.Repository{DB: db, RedisClient: RedisClient} // Redis ditambahkan ke repo jika diperlukan di repo
	app.Use(cors.New(cors.Config{AllowCredentials: false, AllowOrigins: "*"}))
	repo.SetupRoutes(app)

	// Jalankan server
	log.Fatal(app.Listen(":8081"))
}
