package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if os.Getenv("SVP_SKIP_DOTENV") == "1" {
		return
	}
	for _, p := range []string{".env", filepath.Join("..", ".env"), filepath.Join("..", "..", ".env")} {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := godotenv.Load(p); err != nil {
			continue
		}
		log.Printf("loaded env from %s", p)
		return
	}
}
