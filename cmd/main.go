package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/senyabanana/tender-service/internal/db"
	"github.com/senyabanana/tender-service/internal/handlers"
	"github.com/senyabanana/tender-service/internal/repository"
	"github.com/senyabanana/tender-service/internal/router"
	"github.com/senyabanana/tender-service/internal/router/config"
	"github.com/senyabanana/tender-service/internal/services"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	runDBMigration(cfg.MigrationURL, cfg.PostgresConn)

	dbPool, err := db.InitDb(cfg)
	if err != nil {
		log.Fatalf("error initializing database: %v", err)
	}
	defer dbPool.Close()

	logger := log.New(os.Stdout, "INFO: ", log.LstdFlags)

	tenderRepo := repository.NewPostgresTenderRepository(dbPool)
	bidRepo := repository.NewPostgresBidRepository(dbPool)

	tenderService := services.NewTenderService(tenderRepo, dbPool)
	bidService := services.NewBidService(bidRepo, dbPool)

	tenderHandler := handlers.NewTenderHandler(tenderService, logger, 5*time.Second, dbPool)
	bidHandler := handlers.NewBIdHandler(bidService, logger, 5*time.Second, dbPool)

	routes := router.InitRoutes(tenderHandler, bidHandler)

	log.Printf("server is listening on %s...", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, routes); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal("cannot create a new migrate instance", err)
	}

	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("failed to run migrate up:", err)
	}
	log.Println("db migrated successfully")
}
