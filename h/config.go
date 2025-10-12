package h

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// LoadEnv loads environment variables from .env file (if not in production) and processes them into cfg.
// Returns an error if environment variable processing fails.
func LoadEnv(cfg any) error {
	env := os.Getenv("ENV")
	if env != "production" && env != "prod" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Warnf("unable to load .env file: %v", err)
		}
	}
	err := envconfig.Process("", cfg)
	if err != nil {
		return err
	}
	return nil
}

// MustLoadEnv is a convenience wrapper around LoadEnv that panics on error.
// Use this only in main() or initialization code where panic is acceptable.
func MustLoadEnv(cfg any) {
	if err := LoadEnv(cfg); err != nil {
		panic(err)
	}
}
