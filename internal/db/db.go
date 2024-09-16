package db

import (
	"context"
	"fmt"

	"github.com/senyabanana/tender-service/internal/router/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitDb инициализирует подключение к базе данных и возвращает пул соединений.
func InitDb(cfg config.Config) (*pgxpool.Pool, error) {
	dbUser := cfg.PostgresUser
	dbPassword := cfg.PostgresPass
	dbHost := cfg.PostgresHost
	dbPort := cfg.PostgresPort
	dbName := cfg.PostgresDB

	if dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "" {
		return nil, fmt.Errorf("one or more database connection environment variables are missing")
	}

	databaseUrl := cfg.PostgresConn

	var err error
	dbPool, err := pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return dbPool, nil
}
