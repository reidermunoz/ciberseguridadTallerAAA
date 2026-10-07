package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"automarket/audit"
	"automarket/auth"
	"automarket/models"
	"automarket/service"
	"automarket/storage"
)

const (
	sessionTTL        = time.Hour
	minAuditKeyLength = 32
)

type config struct {
	addr          string
	dataDir       string
	auditKey      string
	adminName     string
	adminEmail    string
	adminPassword string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	users, err := storage.OpenTable[models.User](filepath.Join(cfg.dataDir, "users.json"))
	if err != nil {
		return fmt.Errorf("abrir usuarios: %w", err)
	}
	vehicles, err := storage.OpenTable[models.Vehicle](filepath.Join(cfg.dataDir, "vehicles.json"))
	if err != nil {
		return fmt.Errorf("abrir vehículos: %w", err)
	}
	logger, err := audit.Open(filepath.Join(cfg.dataDir, "audit.log"), []byte(cfg.auditKey))
	if err != nil {
		return fmt.Errorf("abrir bitácora: %w", err)
	}
	defer logger.Close()

	accounts := service.NewAccounts(users, vehicles, auth.NewSessions(sessionTTL))
	if err := accounts.EnsureAdmin(service.RegisterInput{
		Name:     cfg.adminName,
		Email:    cfg.adminEmail,
		Password: cfg.adminPassword,
	}); err != nil {
		return fmt.Errorf("crear administrador: %w", err)
	}

	router, err := newRouter(logger, accounts, service.NewVehicles(vehicles, users))
	if err != nil {
		return err
	}
	log.Printf("AutoMarket escuchando en http://%s", cfg.addr)
	return router.Run(cfg.addr)
}

func loadConfig() (config, error) {
	cfg := config{
		addr:          envOr("AUTOMARKET_ADDR", "localhost:8080"),
		dataDir:       envOr("AUTOMARKET_DATA_DIR", "data"),
		auditKey:      os.Getenv("AUTOMARKET_AUDIT_KEY"),
		adminName:     envOr("AUTOMARKET_ADMIN_NAME", "Administrador"),
		adminEmail:    os.Getenv("AUTOMARKET_ADMIN_EMAIL"),
		adminPassword: os.Getenv("AUTOMARKET_ADMIN_PASSWORD"),
	}
	if len(cfg.auditKey) < minAuditKeyLength {
		return cfg, fmt.Errorf("AUTOMARKET_AUDIT_KEY debe tener al menos %d caracteres", minAuditKeyLength)
	}
	if cfg.adminEmail == "" || cfg.adminPassword == "" {
		return cfg, errors.New("AUTOMARKET_ADMIN_EMAIL y AUTOMARKET_ADMIN_PASSWORD son obligatorias")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
