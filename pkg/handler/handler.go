package handler

import (
	"github.com/zikwall/ck-nginx/pkg/nginx"
	"github.com/zikwall/clickhouse-buffer/v3/src/cx"
)

type Handler interface {
	Handle(content string) (cx.Vector, error)
}

type RowHandler struct {
	template   *nginx.Template
	typeCaster nginx.TypeCaster
	columns    []string
}

func (r *RowHandler) Handle(content string) (cx.Vector, error) {
	entry, err := r.template.ParseString(content)
	if err != nil {
		return nil, err
	}
	vector := make(cx.Vector, 0, len(r.columns))
	for _, column := range r.columns {
		value, err := entry.Field(column)
		if err != nil {
			return nil, err
		}
		casted, err := r.typeCaster.TryCast(column, value)
		if err != nil {
			return nil, err
		}
		vector = append(vector, casted)
	}
	return vector, nil
}

func NewRowHandler(columns []string, template *nginx.Template, typeCaster nginx.TypeCaster) *RowHandler {
	return &RowHandler{
		columns:    columns,
		template:   template,
		typeCaster: typeCaster,
	}
}
