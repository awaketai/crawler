package collect

type Tmp struct{
	data map[string]any
}

func (t *Tmp) Get(key string) any{
	return t.data[key]
}

func (t *Tmp) Set(key string,val any) error{
	if t.data == nil {
		t.data = map[string]any{}
	}
	t.data[key] = val

	return nil
}