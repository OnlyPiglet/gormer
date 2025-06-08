package gormer

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"strings"
	"time"
)

type CustomerJsonField map[string]interface{}

func (c CustomerJsonField) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *CustomerJsonField) Scan(input interface{}) error {
	return json.Unmarshal(input.([]byte), c)
}

const customerJsonFieldName = "customer_json_field"

type Model struct {
	ID                uint              `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	DeletedAt         gorm.DeletedAt    `gorm:"index" json:"deleted_at"`
	CustomerJsonField CustomerJsonField `gorm:"type:json" json:"customer_json_field"`
}

func (cjf *CustomerJsonField) MarshalCSV() (string, error) {
	b, err := json.Marshal(cjf)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (cjf *CustomerJsonField) UnmarshalCSV(s string) error {
	return json.Unmarshal([]byte(s), cjf)
}

const customerJsonFieldPrefix = "cjw_"

func IfCustomerJsonField(field string) bool {
	return strings.HasPrefix(field, customerJsonFieldPrefix)
}

// AdvancedWhere 高级查询扩展，类似
type AdvancedWhere struct {
}

type Order int

const (
	DESC Order = iota
	ASC
)

type WhereType int

const (
	NormalType WhereType = iota
	JsonType
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
	PageSize           int                 `json:"page_size"`
	Page               int                 `json:"page"`
	Order              Order               `json:"order"`
	OrderBy            string              `json:"order_by"`
	Wheres             []Where             `json:"wheres"`
	CustomerJsonWheres []Where             `json:"customer_json_where"`
	AdviceItemFuncs    []AdviceItemFunc[T] `json:"advice_item_funcs"`
	Preloads           []string            `json:"preloads"`
	Omits              []string            `json:"omits"`
}

type Where struct {
	Query string      `json:"query"`
	Type  WhereType   `json:"type"`
	Args  interface{} `json:"args"`
}

type QueryListResult[T any] struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Data     []T   `json:"data"`
}

func NewQueryListConfig[T any]() *QueryListConfig[T] {
	return &QueryListConfig[T]{
		PageSize:        999999,
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

func (qc *QueryListConfig[T]) WithOmits(omits []string) *QueryListConfig[T] {
	qc.Omits = append(qc.Omits, omits...)
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
		Total:    0,
		Page:     qc.Page,
		PageSize: qc.PageSize,
		Data:     make([]T, 0),
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

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			tdc = tdc.Preload(preload)
		}
	}

	if qc.Omits != nil && len(qc.Omits) > 0 {
		tdc.Omit(qc.Omits...)
	}

	for _, where := range qc.Wheres {
		switch where.Type {
		case JsonType:
			tdc = tdc.Where(fmt.Sprintf("JSON_EXTRACT(`%s`,'$.%s') like (?)", customerJsonFieldName, where.Query), "%"+where.Args.(string)+"%")
		default:
			tdc = tdc.Where(where.Query, where.Args)
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

type QueryConfig[T any] struct {
	Wheres          []Where             `json:"wheres"`
	Preloads        []string            `json:"preloads"`
	Omits           []string            `json:"omits"`
	AdviceItemFuncs []AdviceItemFunc[T] `json:"advice_item_funcs"`
}

func NewQueryConfig[T any]() *QueryConfig[T] {
	return &QueryConfig[T]{
		Wheres:          []Where{},
		Preloads:        []string{},
		AdviceItemFuncs: make([]AdviceItemFunc[T], 0),
	}
}

func (qc *QueryConfig[T]) WithWheres(wheres []Where) *QueryConfig[T] {
	qc.Wheres = append(qc.Wheres, wheres...)
	return qc
}

func (qc *QueryConfig[T]) WithPreloads(preloads []string) *QueryConfig[T] {
	qc.Preloads = append(qc.Preloads, preloads...)
	return qc
}

func (qc *QueryConfig[T]) WithOmits(omits []string) *QueryConfig[T] {
	qc.Omits = append(qc.Omits, omits...)
	return qc
}

func (qc *QueryConfig[T]) WithAdviceItemFunc(funcs []AdviceItemFunc[T]) *QueryConfig[T] {
	qc.AdviceItemFuncs = append(qc.AdviceItemFuncs, funcs...)
	return qc
}

func Exist[T any](db *gorm.DB, qc *QueryConfig[T]) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("get db client failed")
	}

	db = db.Model(*new(T))

	if qc != nil && qc.Wheres != nil {
		for _, where := range qc.Wheres {
			switch where.Type {
			case JsonType:
				db = db.Where(fmt.Sprintf("JSON_EXTRACT(`%s`,'$.%s') like (?)", customerJsonFieldName, where.Query), "%"+where.Args.(string)+"%")
			default:
				db = db.Where(where.Query, where.Args)
			}
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

func Delete[T any](db *gorm.DB, qc *QueryConfig[T]) error {
	if db == nil {
		return fmt.Errorf("get db client failed")
	}

	t, err := Query[T](db, qc)

	if err != nil {
		return err
	}

	return db.Delete(t).Error
}

// BatchUpdate 批量更新，如存在错误，会回滚所有批量操作
func BatchUpdate[T any](db *gorm.DB, records []T) error {
	tx := db.Begin()
	for _, record := range records {
		err := Update[T](tx, &record)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// BatchCreate 批量创建，如存在错误，会回滚所有批量操作
func BatchCreate[T any](db *gorm.DB, records []T) error {
	tx := db.Begin()
	for _, record := range records {
		err := Create[T](tx, record)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// BatchDelete 批量删除，如存在错误，会回滚所有批量操作
func BatchDelete[T any](db *gorm.DB, records []T) error {
	tx := db.Begin()
	for _, record := range records {
		err := tx.Delete(&record).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func QueryWithNotFoundErr[T any](db *gorm.DB, qc *QueryConfig[T]) (*T, error) {
	if db == nil {
		return nil, fmt.Errorf("get db client failed")
	}

	db = db.Model(*new(T))

	if qc.Omits != nil && len(qc.Omits) > 0 {
		db = db.Omit(qc.Omits...)
	}

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			db = db.Preload(preload)
		}
	}

	if qc != nil && qc.Wheres != nil {
		for _, where := range qc.Wheres {
			switch where.Type {
			case JsonType:
				db = db.Where(fmt.Sprintf("JSON_EXTRACT(`%s`,'$.%s') like (?)", customerJsonFieldName, where.Query), "%"+where.Args.(string)+"%")
			default:
				db = db.Where(where.Query, where.Args)
			}
		}
	}

	t := new(T)

	e := db.First(t).Error

	if e != nil {
		return nil, e
	}

	for _, itemFunc := range qc.AdviceItemFuncs {
		var err error
		*t, err = itemFunc(*t)
		if err != nil {
			slog.Warn(err.Error())
		}
	}

	return t, nil

}

func Query[T any](db *gorm.DB, qc *QueryConfig[T]) (*T, error) {

	if db == nil {
		return nil, fmt.Errorf("get db client failed")
	}

	db = db.Model(*new(T))

	if qc.Omits != nil && len(qc.Omits) > 0 {
		db = db.Omit(qc.Omits...)
	}

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			db = db.Preload(preload)
		}
	}

	if qc != nil && qc.Wheres != nil {
		for _, where := range qc.Wheres {
			switch where.Type {
			case JsonType:
				db = db.Where(fmt.Sprintf("JSON_EXTRACT(`%s`,'$.%s') like (?)", customerJsonFieldName, where.Query), "%"+where.Args.(string)+"%")
			default:
				db = db.Where(where.Query, where.Args)
			}
		}
	}

	t := new(T)

	e := db.First(t).Error

	if e != nil && errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil

	} else if e != nil {
		return nil, e

	}

	for _, itemFunc := range qc.AdviceItemFuncs {
		var err error
		*t, err = itemFunc(*t)
		if err != nil {
			slog.Warn(err.Error())
		}
	}

	return t, nil

}

func EntityContentById[T any](dbe *gorm.DB, dbq *gorm.DB, id string) (string, error) {
	qc := NewQueryConfig[T]().WithWheres([]Where{
		{
			Query: "id = ?",
			Args:  id,
		},
	})

	exist, err := Exist[T](dbe, qc)

	if err != nil {
		return "", err
	}

	if !exist {
		return "", nil
	}

	dictData, err := Query[T](dbq, qc)

	if err != nil {
		return "", err
	}

	marshal, err := json.Marshal(dictData)

	if err != nil {
		return "", err
	}

	return string(marshal), nil
}

func QueryAll[T any](tdc *gorm.DB, qc *QueryListConfig[T]) ([]T, error) {

	qr := &QueryListResult[T]{
		Total: 0,
		Data:  make([]T, 0),
	}

	tdc = tdc.Model(*new(T))

	if qc != nil && qc.Preloads != nil {
		for _, preload := range qc.Preloads {
			tdc = tdc.Preload(preload)
		}
	}

	if qc.Omits != nil && len(qc.Omits) > 0 {
		tdc.Omit(qc.Omits...)
	}

	for _, where := range qc.Wheres {
		switch where.Type {
		case JsonType:
			tdc = tdc.Where(fmt.Sprintf("JSON_EXTRACT(`%s`,'$.%s') like (?)", customerJsonFieldName, where.Query), "%"+where.Args.(string)+"%")
		default:
			tdc = tdc.Where(where.Query, where.Args)
		}
	}

	err := tdc.Order(fmt.Sprintf("%s %s", qc.OrderBy, qc.Order.String())).Find(&qr.Data).Error

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

	return qr.Data, nil

}
