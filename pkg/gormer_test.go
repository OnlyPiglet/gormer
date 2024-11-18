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
	//ddb := db.WithContext(context.Background())
	//query, err := Query[User](ddb, NewQueryConfig().WithWheres([]Where{
	//	{
	//		"gorm_cjw_nb",
	//		JsonType,
	//		"78",
	//	},
	//}))
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("%v", query)
}

func TestCreateJsonField(t *testing.T) {
	if db == nil {
		log.Fatalf("db is nil")
	}
	ddb := db.WithContext(context.Background())
	u := User{
		Model: Model{
			CustomerJsonField: map[string]interface{}{
				"gorm_cjw_na": "asd",
				"gorm_cjw_nb": 456,
			},
		},
		UserName: "123",
		Password: "456",
	}
	err := Create[User](ddb, u)
	if err != nil {
		panic(err)
	}
}

func TestBatchCreate(t *testing.T) {
	Create[[]User](db.WithContext(context.Background()), []User{{
		UserName: "123",
		Password: "456",
	}, {
		UserName: "123123",
		Password: "456123",
	}})
}

func TestBatchUpdate(t *testing.T) {
	list, err := QueryList[User](db.WithContext(context.Background()), db.WithContext(context.Background()), NewQueryListConfig[User]().WithPage(1).WithPageSize(100000))
	if err != nil {
		panic(err)
	}
	for i, datum := range list.Data {
		datum.UserName = "123change" + datum.UserName
		list.Data[i] = datum
		log.Printf("%+v", list.Data[i])
	}
	list.Data = append(list.Data, User{
		Model: Model{
			ID: 90,
		},
		UserName: "a123change" + list.Data[0].UserName,
		Password: "123change" + list.Data[0].Password,
	})
	err = BatchUpdate[User](db.WithContext(context.Background()), list.Data)
	if err != nil {
		panic(err)
	}
}

func TestBatchDelete(t *testing.T) {
	d := db.WithContext(context.Background())
	d.Begin()
	err := d.Delete([]User{
		{
			Model: Model{
				ID: 11,
			},
		},
		{
			Model: Model{
				ID: 6,
			},
		},
	}).Error
	if err != nil {
		println(err.Error())
		d.Callback()
	} else {
		d.Commit()
	}
}
