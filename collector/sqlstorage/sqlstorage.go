package sqlstorage

import (
	"encoding/json"
	"fmt"

	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/engine"
	"github.com/awaketai/crawler/sqldb"
	"go.uber.org/zap"
)

type SqlStore struct {
	// 分匹输出结果缓存
	dataDocker  []*collector.DataCell
	columnNames []sqldb.Field
	db          sqldb.DBer
	Table       map[string]struct{}
	options
}

func NewSqlStore(opts ...Option) (*SqlStore, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &SqlStore{}
	s.options = options
	s.Table = map[string]struct{}{}
	db, err := sqldb.NewSqlDB(
		sqldb.WithDSN(s.dsn),
		sqldb.WithLogger(s.logger),
	)
	if err != nil {
		return nil, err
	}
	s.db = db

	return s, nil
}

func (s *SqlStore) Save(dataCells ...*collector.DataCell) error {
	for _, cell := range dataCells {
		tableName := cell.GetTableName()
		s.logger.Info("save data to table", zap.String("table", tableName))
		if _, ok := s.Table[tableName]; !ok {
			// create table
			columnNames := s.getFields(cell)
			err := s.db.CreateTable(sqldb.TableData{
				TableName:   tableName,
				ColumnNames: columnNames,
				AutoKey:     true,
			})
			if err != nil {
				s.logger.Error("create table failed", zap.Error(err))
			}
			s.Table[tableName] = struct{}{}
		}
		s.dataDocker = append(s.dataDocker, cell)
		if len(s.dataDocker) >= s.BatchCount {
			s.Flush()
		}
	}

	return nil
}

func (s *SqlStore) getFields(cell *collector.DataCell) []sqldb.Field {
	taskName := cell.Data["Task"].(string)
	ruleName := cell.Data["Rule"].(string)
	fields := engine.GetFields(taskName, ruleName)
	var columnNames []sqldb.Field
	for _, field := range fields {
		columnNames = append(columnNames, sqldb.Field{
			Title: field,
			Type:  "MEDIUMTEXT",
		})
	}
	columnNames = append(columnNames, sqldb.Field{
		Title: "Url",
		Type:  "VARCHAR(250)",
	}, sqldb.Field{
		Title: "Time",
		Type:  "VARCHAR(255)",
	})

	return columnNames
}

func (s *SqlStore) Flush() error {
	if len(s.dataDocker) == 0 {
		return nil
	}
	args := make([]any, 0, len(s.dataDocker))
	for _, dataCell := range s.dataDocker {
		ruleName := dataCell.Data["Rule"].(string)
		taskName := dataCell.Data["Task"].(string)
		fields := engine.GetFields(taskName, ruleName)
		data := dataCell.Data["Data"].(map[string]any)
		value := []string{}
		for _, field := range fields {
			v := data[field]
			switch v := v.(type) {
			case nil:
				value = append(value, "")
			case string:
				value = append(value, v)
			default:
				j, err := json.Marshal(v)
				if err != nil {
					s.logger.Error("marshal file err", zap.Error(err))
					return err
				}
				value = append(value, string(j))
			}
		}
		value = append(value,
			dataCell.Data["Url"].(string),
			dataCell.Data["Time"].(string),
		)
		for _, v := range value {
			args = append(args, v)
		}
	}
	fmt.Println("args:", args)

	err := s.db.Insert(sqldb.TableData{
		TableName:   s.dataDocker[0].GetTableName(),
		ColumnNames: s.getFields(s.dataDocker[0]),
		Args:        args,
		DataCount:   len(s.dataDocker),
	})
	if err != nil {
		s.logger.Error("insert data failed", zap.Error(err))
	}

	return nil
}
