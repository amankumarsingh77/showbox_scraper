package main

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	Server ServerConfig
	Mongo  MongoConfig
}

type ServerConfig struct {
	Port string
}

type MongoConfig struct {
	URI string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Server: ServerConfig{
			Port: os.Getenv("PORT"),
		},
		Mongo: MongoConfig{
			URI: os.Getenv("MONGO_URI"),
		},
	}
	return cfg, nil
}
