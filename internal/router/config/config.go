package config

import "github.com/spf13/viper"

// Config - структура для хранения конфигураций приложения
type Config struct {
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
	PostgresConn  string `mapstructure:"POSTGRES_CONN"`
	PostgresURL   string `mapstructure:"POSTGRES_JDBC_URL"`
	PostgresUser  string `mapstructure:"POSTGRES_USERNAME"`
	PostgresPass  string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresHost  string `mapstructure:"POSTGRES_HOST"`
	PostgresPort  string `mapstructure:"POSTGRES_PORT"`
	PostgresDB    string `mapstructure:"POSTGRES_DATABASE"`
	MigrationURL  string `mapstructure:"MIGRATION_URL"`
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (cfg Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&cfg)
	return
}
