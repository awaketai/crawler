package sqldb

import (
	"database/sql"
	"errors"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type DBer interface {
	CreateTable(TableData) error
	Insert(TableData) error
}

type SqlDB struct {
	options
	db *sql.DB
}

func new() *SqlDB {
	return &SqlDB{}
}

type Field struct {
	Title string
	Type  string
}

type TableData struct {
	TableName   string
	ColumnNames []Field
	Args        []any
	DataCount   int
	AutoKey     bool
}

func NewSqlDB(opts ...Option) (*SqlDB, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	db := new()
	db.options = options
	if err := db.OpenDB(); err != nil {
		return nil, err
	}

	return db, nil
}

func (s *SqlDB) OpenDB() error {
	db, err := sql.Open("mysql", s.dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	if err := db.Ping(); err != nil {
		return err
	}

	s.db = db
	return nil
}

func (s *SqlDB) CreateTable(t TableData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("column can not be empty")
	}
	sql := `CREATE TABLE IF NOT EXISTS ` + t.TableName + " ("
	if t.AutoKey {
		sql += `id INT(12) NOT NULL PRIMARY KEY AUTO_INCREMENT,`
	}
	for _, t := range t.ColumnNames {
		sql += t.Title + ` ` + t.Type + `,`
	}
	sql = sql[:len(sql)-1] + `) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
	s.logger.Debug("create table", zap.String("sql", sql))
	_, err := s.db.Exec(sql)

	return err
}

func (s *SqlDB) Insert(t TableData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("empty column")
	}
	sql := `INSERT INTO ` + t.TableName + `(`
	for _, v := range t.ColumnNames {
		sql += v.Title + ","
	}
	sql = sql[:len(sql)-1] + `) VALUES `
	blank := ",(" + strings.Repeat(".?", len(t.ColumnNames))[1:] + ")"
	sql += strings.Repeat(blank, t.DataCount)[1:] + `;`
	s.logger.Debug("insert table", zap.String("sql", sql))
	_, err := s.db.Exec(sql, t.Args...)

	return err
}
