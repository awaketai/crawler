package collector

type Storager interface{
	Save(datas ...*DataCell) error
}

type DataCell struct{
	Data map[string]any
}

func (d *DataCell) GetTableName() string {
	return d.Data["Task"].(string)
}

func (d *DataCell) GetTaskName() string {
	return d.Data["Task"].(string)
}

