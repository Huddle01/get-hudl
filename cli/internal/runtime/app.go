package runtime

import (
	"context"
	"io"
	"time"

	"github.com/Huddle01/get-hudl/cli/internal/config"
)

type contextKey struct{}

type GlobalOptions struct {
	APIKey    string
	Workspace string
	Region    string
	Output    string
	CloudBase string
	GPUBase   string
	Timeout   time.Duration
	Verbose   bool
	NoColor   bool
	Quiet     bool
}

type MutateOptions struct {
	File           string
	IdempotencyKey string
	DryRun         bool
	Interactive    bool
	Yes            bool
}

type App struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Options  GlobalOptions
	Config   config.Resolved
	HTTP     *Client
	IsTTYOut bool
	IsTTYIn  bool
}

func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, contextKey{}, app)
}

func FromContext(ctx context.Context) *App {
	if app, ok := ctx.Value(contextKey{}).(*App); ok {
		return app
	}
	return nil
}
