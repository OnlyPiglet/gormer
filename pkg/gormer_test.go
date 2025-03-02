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
	BatchCreate[User](db.WithContext(context.Background()), []User{
		{UserName: "123",
			Password: "456"},
		{UserName: "asd123",
			Password: "asd456"},
	})
}

func TestGetUser(t *testing.T) {

	list, err := QueryList[User](db.WithContext(context.Background()), db.WithContext(context.Background()), NewQueryListConfig[User]().WithWheres(
		[]Where{
			{
				Query: "user_name = ?",
				Args:  "aaaa123changeaasd123changea123change123changechange123",
			},
		}).WithOmits([]string{"UserName"}))

	if err != nil {
		panic(err)
	}
	println(list.Data[0].Password)
}

func TestBatchUpdate(t *testing.T) {
	list, err := QueryList[User](db.WithContext(context.Background()), db.WithContext(context.Background()), NewQueryListConfig[User]().WithPage(1).WithPageSize(100000))
	if err != nil {
		panic(err)
	}
	for i, datum := range list.Data {
		datum.UserName = "aaaa123change" + datum.UserName
		list.Data[i] = datum
		log.Printf("%+v", list.Data[i])
	}
	err = BatchUpdate[User](db.WithContext(context.Background()), list.Data)
	if err != nil {
		panic(err)
	}
}

func TestBatchDelete(t *testing.T) {
	BatchDelete[User](db.WithContext(context.Background()), []User{{Model: Model{ID: 90}}, {Model: Model{ID: 91}}})
}

type B struct {
	Model
	Username string `json:"username"`
	Password string `json:"password"`
}

func (b *B) TableName() string {
	return "b"
}

type A struct {
	Model
	Bid uint
	B   B `gorm:"foreignKey:Bid;references:ID"`
}

func (a *A) TableName() string {
	return "a"
}

func TestKey(t *testing.T) {
	testing.Init()

	db.Exec("drop table a;")
	db.Exec("drop table b;")
	db.Migrator().AutoMigrate(A{}, B{})
	//db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(A{}).Delete(B{})

	db.Save(&B{
		Model:    Model{ID: 3},
		Username: "3",
		Password: "3",
	})

	db.Save(&B{
		Model:    Model{ID: 2},
		Username: "2",
		Password: "2",
	})
	a := &A{
		Bid: 3,
		B: B{
			Model:    Model{ID: 3},
			Username: "2",
			Password: "2",
		},
	}

	db.Session(&gorm.Session{
		//此字段 true 则会联机更新 子对象的属性，否则只更新外键
		FullSaveAssociations: false,
	}).Save(a)

}
