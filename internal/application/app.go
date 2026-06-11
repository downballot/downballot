package application

import (
	"context"

	"gorm.io/gorm"
)

// App is a structure that contains the relevant services.
type App struct {
	db *gorm.DB
}

// New creates a new app.
func New(ctx context.Context, db *gorm.DB) *App {
	a := &App{
		db: db.Session(&gorm.Session{}),
	}
	return a
}

// DB returns a fresh database handle.
func (a *App) DB() *gorm.DB {
	return a.db.Session(&gorm.Session{})
}
