package gormer

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
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

type QueryListConfig[T any] struct {
	PageSize        int                 `json:"page_size"`
	Page            int                 `json:"page"`
	Order           Order               `json:"order"`
	OrderBy         string              `json:"order_by"`
	Wheres          []Where             `json:"wheres"`
	AdviceItemFuncs []AdviceItemFunc[T] `json:"advice_item_funcs"`
	Preloads        []string            `json:"preloads"`
}

type Where struct {
	Query string      `json:"query"`
	Args  interface{} `json:"args"`
}

type QueryListResult[T any] struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Data  []T   `json:"data"`
}

func NewQueryListConfig[T any]() *QueryListConfig[T] {
	return &QueryListConfig[T]{
		PageSize:        10,
		Page:            1,
		Order:           DESC,
		OrderBy:         "updated_at",
		Wheres:          []Where{},
		AdviceItemFuncs: []AdviceItemFunc[T]{},
		Preloads:        []string{},
	}
}

type AdviceItemFunc[T any] func(t T) (T, error)

func (qc *QueryListConfig[T]) WithWheres(wheres []Where) *QueryListConfig[T] {
	qc.Wheres = append(qc.Wheres, wheres...)
	return qc
}

func (qc *QueryListConfig[T]) WithPreloads(preloads []string) *QueryListConfig[T] {
	qc.Preloads = append(qc.Preloads, preloads...)
	return qc
}

func (qc *QueryListConfig[T]) WithAdviceItemFunc(funcs []AdviceItemFunc[T]) *QueryListConfig[T] {
	qc.AdviceItemFuncs = append(qc.AdviceItemFuncs, funcs...)
	return qc
}

func (qc *QueryListConfig[T]) WithOrderBy(orderBy string) *QueryListConfig[T] {
	qc.OrderBy = orderBy
	return qc
}

func (qc *QueryListConfig[T]) WithOrder(order Order) *QueryListConfig[T] {
	qc.Order = order
	return qc
}

func (qc *QueryListConfig[T]) WithPageSize(pageSize int) *QueryListConfig[T] {
	if pageSize <= 0 {
		return qc
	}
	qc.PageSize = pageSize
	return qc
}

func (qc *QueryListConfig[T]) WithPage(page int) *QueryListConfig[T] {
	if page <= 0 {
		return qc
	}
	qc.Page = page
	return qc
}

func QueryList[T any](dc *gorm.DB, tdc *gorm.DB, qc *QueryListConfig[T]) (*QueryListResult[T], error) {

	qr := &QueryListResult[T]{
		Total: 0,
		Page:  qc.Page,
		Data:  make([]T, 0),
	}

	dc = dc.Model(*new(T))

	for _, where := range qc.Wheres {
		dc = dc.Where(where.Query, where.Args)
	}

	if err := dc.Count(&qr.Total).Error; err != nil {
		return nil, err
	}

	offset := (qc.Page - 1) * qc.PageSize

	tdc = tdc.Model(*new(T))
	for _, where := range qc.Wheres {
		tdc = tdc.Where(where.Query, where.Args)
	}

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			tdc = tdc.Preload(preload)
		}
	}

	err := tdc.Order(fmt.Sprintf("%s %s", qc.OrderBy, qc.Order.String())).Offset(offset).Limit(qc.PageSize).Find(&qr.Data).Error

	if err != nil {
		return nil, err
	}

	for _, itemFunc := range qc.AdviceItemFuncs {
		for j, datum := range qr.Data {
			qr.Data[j], err = itemFunc(datum)
			if err != nil {
				slog.Warn(err.Error())
			}
		}
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
	Wheres   []Where  `json:"wheres"`
	Preloads []string `json:"preloads"`
}

func NewQueryConfig() *QueryConfig {
	return &QueryConfig{
		Wheres:   []Where{},
		Preloads: []string{}}
}

func (qc *QueryConfig) WithWheres(wheres []Where) *QueryConfig {
	qc.Wheres = append(qc.Wheres, wheres...)
	return qc
}

func (qc *QueryConfig) WithPreloads(preloads []string) *QueryConfig {
	qc.Preloads = append(qc.Preloads, preloads...)
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

	if err := db.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil

}

func Update[T any](db *gorm.DB, t *T) error {
	if db == nil {
		return fmt.Errorf("get db client failed")
	}
	return db.Save(t).Error
}

func Delete[T any](db *gorm.DB, qc *QueryConfig) error {
	if db == nil {
		return fmt.Errorf("get db client failed")
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

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			db = db.Preload(preload)
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
