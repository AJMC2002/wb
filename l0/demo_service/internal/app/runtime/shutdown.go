package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func NotifyContext(parent context.Context) (context.Context, func()) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}
