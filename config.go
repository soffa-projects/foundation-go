package micro

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

func LoadConfig(cfg any) {
	env := os.Getenv("ENV")
	if env != "production" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Warnf("unable to load .env file: %v", err)
		}
	}
	err := envconfig.Process("", cfg)
	if err != nil {
		log.Fatal(err)
	}
}
