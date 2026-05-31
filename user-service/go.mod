module short-video-platform/user-service

go 1.22.0

require (
	github.com/joho/godotenv v1.5.1
	golang.org/x/crypto v0.32.0
	google.golang.org/grpc v1.71.0
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
	short-video-platform/gen v0.0.0
	short-video-platform/pkg/auth v0.0.0
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace (
	short-video-platform/gen => ../gen
	short-video-platform/pkg/auth => ../pkg/auth
)
