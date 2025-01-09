Initiate the Go project

mkdir event-analytics
cd event-analytics
go mod init event-analytics
go mod init github.com/tulbadex/event-analytics

Install required dependencies

go get -u github.com/gin-gonic/gin
go get -u github.com/jmoiron/sqlx # For database handling
go get -u github.com/go-redis/redis/v8
go get -u github.com/segmentio/kafka-go # Optional: If using Kafka
go get github.com/gin-gonic/gin github.com/go-redis/redis/v8 github.com/jackc/pgx/v5 golang.org/x/crypto/bcrypt
go get -u gorm.io/gorm # for migration
go get -u gorm.io/driver/postgres # for postgres
go get github.com/joho/godotenv # for .env
go get gopkg.in/gomail.v2 # for email
go get github.com/google/uuid # for uuid
go get github.com/go-co-op/gocron # for cron
go get github.com/gorilla/websocket # for websocket
go get github.com/stretchr/testify/assert # for test assertions
go get -u gorm.io/driver/sqlite # for sqlite


# for existing project
## you need to initialize a Go module if you haven't already
go mod init event-analytics
## After setting up the go.mod file, run
go mod tidy
## This will download all required dependencies and fix the module path issues.
go list