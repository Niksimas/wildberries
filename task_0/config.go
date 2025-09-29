package main

import (
	"os"
)

// Config - структура для хранения конфигурации
type Config struct {
	DatabaseURL string
	Port        string
}

// getConfig - получение конфигурации из переменных окружения
func getConfig() *Config {
	config := &Config{
		// Значения по умолчанию
		DatabaseURL: "user=program dbname=wildberries sslmode=disable password=Vny-1MHyZnSf host=localhost",
		Port:        ":8080",
	}

	// Получаем значения из переменных окружения, если они заданы
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}

	if port := os.Getenv("PORT"); port != "" {
		config.Port = ":" + port
	}

	return config
}
