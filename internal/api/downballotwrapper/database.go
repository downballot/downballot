package downballotwrapper

import (
	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
)

// getDatabaseFromRequest returns the database from the request.
//
// If no database is found, then nil is returned.
func getDatabaseFromRequest(req *restful.Request) *gorm.DB {
	dbValue := req.Attribute(attributeDatabase)
	if dbValue == nil {
		return nil
	}
	db := dbValue.(*gorm.DB)
	return db.Session(&gorm.Session{}).WithContext(req.Request.Context())
}

func setDatabaseForRequest(req *restful.Request, db *gorm.DB) {
	req.SetAttribute(attributeDatabase, db)
}
