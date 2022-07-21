package handler

import (
	"github.com/zikwall/ck-nginx/pkg/nginx"
	"github.com/zikwall/clickhouse-buffer/v3/src/cx"
)

type Handler interface {
	Handle(content string) (cx.Vector, error)
}

type RowHandler struct {
	template              *nginx.Template
	typeCaster            nginx.TypeCaster
	rewriteNginxLocalTime bool
}

func (r *RowHandler) Handle(content string) (cx.Vector, error) {
	entry, err := r.template.ParseString(content)
	if err != nil {
		return nil, err
	}
	fields := entry.Fields()
	vector := make(cx.Vector, 0, len(fields))
	for key, value := range fields {
		casted, err := r.typeCaster.TryCast(key, value)
		if err != nil {
			return nil, err
		}
		vector = append(vector, casted)
	}
	return vector, nil
}

func NewRowHandler(rewriteTime bool) *RowHandler {
	return &RowHandler{rewriteNginxLocalTime: rewriteTime}
}
