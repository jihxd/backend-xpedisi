package repository

import (
	"context"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type Repository struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	Ctx         context.Context
}
