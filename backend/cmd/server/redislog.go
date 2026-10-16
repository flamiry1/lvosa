package main

import (
	"context"
	"fmt"
	"log/slog"
)

type slogAdapter struct{}

func (slogAdapter) Printf(ctx context.Context, format string, v ...interface{}) {
	slog.WarnContext(ctx, fmt.Sprintf(format, v...))

}
