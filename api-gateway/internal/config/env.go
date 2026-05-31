package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv 从项目根目录或当前目录加载 .env（本地 go run 时使用；Docker 由 compose 注入）。
func LoadEnv() {
	if os.Getenv("SVP_SKIP_DOTENV") == "1" {
		return
	}

	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := godotenv.Load(p); err != nil {
			log.Printf("load %s: %v", p, err)
			continue
		}
		log.Printf("loaded env from %s", p)
		return
	}
}
