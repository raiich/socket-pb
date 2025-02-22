package log

import (
	"context"
	"log/slog"
)

type Handler struct {
	slog.Handler
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	type hasAttrs interface {
		Attrs() []slog.Attr
	}
	if attrCtx, ok := ctx.(hasAttrs); ok {
		record.AddAttrs(attrCtx.Attrs()...)
	}
	return h.Handler.Handle(ctx, record)
}
