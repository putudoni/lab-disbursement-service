package server

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lab-disbursement-service/internal/model"
)

type gormConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

func (c gormConfig) dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode, c.TimeZone,
	)
}

type dbHolder struct {
	mu sync.Mutex
	db *gorm.DB
}

func newDBHolder() *dbHolder {
	return &dbHolder{}
}

func (h *dbHolder) Init(cfg gormConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.db != nil {
		return nil
	}

	if cfg.Name == "" {
		return errors.New("database name is not configured")
	}

	db, err := gorm.Open(postgres.Open(cfg.dsn()), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(&model.BankAccount{}); err != nil {
		return err
	}

	h.db = db
	return nil
}

func (h *dbHolder) DB() *gorm.DB {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.db
}