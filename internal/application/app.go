package application

import (
	"context"
	"log/slog"

	"github.com/downballot/downballot/internal/cache"
	"gorm.io/gorm"
)

// App is a structure that contains the relevant services.
type App struct {
	Cache *cache.Cache
	db    *gorm.DB
}

// New creates a new app.
func New(ctx context.Context, db *gorm.DB) *App {
	a := &App{
		db: db.Session(&gorm.Session{}),
	}

	var err error
	a.Cache, err = cache.New(10 * 1000 * 1000) // Use 10MB as the maximum cache size.
	if err != nil {
		slog.ErrorContext(ctx, "Could not create the cache.", "err", err) // We don't have a context here.  This happens at initialization.
		panic(err)
	}

	return a
}

// DB returns a fresh database handle.
func (a *App) DB() *gorm.DB {
	return a.db.Session(&gorm.Session{})
}
