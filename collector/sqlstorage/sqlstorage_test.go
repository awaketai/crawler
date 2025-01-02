package sqlstorage

import (
	"testing"

	"github.com/awaketai/crawler/collector"
	"github.com/awaketai/crawler/sqldb"
	"github.com/stretchr/testify/assert"
)

type mysqldb struct {
}

func (m *mysqldb) CreateTable(t sqldb.TableData) error {
	return nil
}

func (m *mysqldb) Insert(t sqldb.TableData) error {
	return nil
}

func TestSQLStorage(t *testing.T) {
	type fields struct {
		dataDocker []*collector.DataCell
		options    options
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "empty",
			fields:  fields{},
			wantErr: false,
		},
		{
			name: "no rule field",
			fields: fields{
				dataDocker: []*collector.DataCell{
					{
						Data: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add your test logic here
			s := &SqlStore{
				dataDocker: tt.fields.dataDocker,
				db:         &mysqldb{},
				options:    tt.fields.options,
			}
			if err := s.Flush(); (err != nil) != tt.wantErr {
				t.Errorf("SqlStore.Flush() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Nil(t, s.dataDocker)
		})
	}
}
