package application

import (
	"github.com/downballot/downballot/internal/cache"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// App is a structure that contains the relevant services.
type App struct {
	Cache *cache.Cache
	DB    *gorm.DB
}

// New creates a new app.
func New() *App {
	a := &App{}

	var err error
	a.Cache, err = cache.New(10 * 1000 * 1000) // Use 10MB as the maximum cache size.
	if err != nil {
		logrus.Errorf("Could not create the cache: %v", err) // We don't have a context here.  This happens at initialization.
		panic(err)
	}

	return a
}
