package gormer

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Order int

const (
	DESC Order = iota
	ASC
)

func (o Order) String() string {
	switch o {
	case ASC:
		return "asc"
	default:
		return "desc"
	}
}

type QueryListConfig struct {
	PageSize int     `json:"page_size"`
	Page     int     `json:"page"`
	Order    Order   `json:"order"`
	OrderBy  string  `json:"order_by"`
	Wheres   []Where `json:"wheres"`
}

type Where struct {
	Query string `json:"query"`
	Args  string `json:"args"`
}

type QueryListResult[T any] struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Data  []T   `json:"data"`
}

var defaultQuery = &QueryListConfig{
	PageSize: 10,
	Page:     1,
	Order:    DESC,
	OrderBy:  "updated_at",
	Wheres:   []Where{},
}

func NewQueryListConfig() *QueryListConfig {
	return defaultQuery
}

func (qc *QueryListConfig) WithWheres(wheres []Where) *QueryListConfig {
	qc.Wheres = append(qc.Wheres, wheres...)
	return qc
}

func (qc *QueryListConfig) WithOrderBy(orderBy string) *QueryListConfig {
	qc.OrderBy = orderBy
	return qc
}

func (qc *QueryListConfig) WithOrder(order Order) *QueryListConfig {
	qc.Order = order
	return qc
}

func (qc *QueryListConfig) WithPageSize(pageSize int) *QueryListConfig {
	if pageSize <= 0 {
		return qc
	}
	qc.PageSize = pageSize
	return qc
}

func (qc *QueryListConfig) WithPage(page int) *QueryListConfig {
	if page <= 0 {
		return qc
	}
	qc.Page = page
	return qc
}

func QueryList[T any](dc *gorm.DB, tdc *gorm.DB, qc *QueryListConfig) (*QueryListResult[T], error) {

	qr := &QueryListResult[T]{
		Total: 0,
		Page:  qc.Page,
		Data:  make([]T, 0),
	}

	tdc = tdc.Model(*new(T))

	for _, where := range qc.Wheres {
		tdc = tdc.Where(where.Query, where.Args)
	}

	if err := tdc.Count(&qr.Total).Error; err != nil {
		return nil, err
	}

	offset := (qc.Page - 1) * qc.PageSize

	dc = dc.Model(*new(T))
	for _, where := range qc.Wheres {
		dc = dc.Where(where.Query, where.Args)
	}

	err := dc.Order(fmt.Sprintf("%s %s", qc.OrderBy, qc.Order.String())).Offset(offset).Limit(qc.PageSize).Find(&qr.Data).Error

	if err != nil {
		return nil, err
	}

	return qr, nil

}

func Create[T any](db *gorm.DB, t T) error {

	if db == nil {
		return fmt.Errorf("get db client failed")
	}

	return db.Model(*new(T)).Create(&t).Error

}

type QueryConfig struct {
	Wheres []Where `json:"wheres"`
}

var defaultQueryConfig = &QueryConfig{
	Wheres: []Where{},
}

func NewQueryConfig() *QueryConfig {
	return defaultQueryConfig
}

func (qc *QueryConfig) WithWheres(wheres []Where) *QueryConfig {
	qc.Wheres = append(qc.Wheres, wheres...)
	return qc
}

func Exist[T any](db *gorm.DB, qc *QueryConfig) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("get db client failed")
	}

	db = db.Model(*new(T))

	if qc != nil && qc.Wheres != nil {
		for _, where := range qc.Wheres {
			db = db.Where(where.Query, where.Args)
		}
	}

	count := int64(0)

	db.Count(&count)

	return count > 0, nil

}

func Update[T any](db *gorm.DB, t T) error {
	if db == nil {
		return fmt.Errorf("get db client failed")
	}
	return db.Model(*new(T)).Save(&t).Error
}

func Delete[T any](db *gorm.DB, qc *QueryConfig) error {
	if db == nil {
		return fmt.Errorf("get db client failed")
	}
	exist, err := Exist[T](db, qc)

	if err != nil {
		return err
	}

	if !exist {
		return nil
	}

	t, err := Query[T](db, qc)

	if err != nil {
		return err
	}

	return db.Delete(t).Error
}

func Query[T any](db *gorm.DB, qc *QueryConfig) (*T, error) {

	if db == nil {
		return nil, fmt.Errorf("get db client failed")
	}

	db = db.Model(*new(T))

	if qc != nil && qc.Wheres != nil {
		for _, where := range qc.Wheres {
			db = db.Where(where.Query, where.Args)
		}
	}

	t := new(T)

	e := db.First(t).Error

	if e != nil && errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil

	} else if e != nil {
		return nil, e

	}

	return t, nil

}
