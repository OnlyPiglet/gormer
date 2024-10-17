package gormer

import (
	"context"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

var db *gorm.DB

func init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", "root", "123456", "localhost", "3306", "test")
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // Don't include params in the SQL log
			Colorful:                  false,       // Disable color
		},
	)
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger})

	if err != nil {
		log.Fatalf("Database connection failed. Database name: %s", "test")
	} else {
		sqlDb, err := db.DB()
		if err != nil {
			panic("初始化数据库失败：" + err.Error())
		}
		sqlDb.SetMaxOpenConns(10)
		sqlDb.SetMaxIdleConns(5)
		sqlDb.SetConnMaxLifetime(30 * time.Second)
	}

	db.Migrator().AutoMigrate(&User{})
}

type User struct {
	Model
	UserName string `json:"user_name"`
	Password string `json:"password"`
}

func TestJsonField(t *testing.T) {
	if db == nil {
		log.Fatalf("db is nil")
	}
	ddb := db.WithContext(context.Background())
	query, err := Query[User](ddb, NewQueryConfig().WithWheres([]Where{
		{
			"gorm_cjw_na",
			JsonType,
			"123",
		},
	}))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", query)
}
