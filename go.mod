module github.com/downballot/downballot

go 1.25.4

require (
	github.com/DmitriyVTitov/size v1.1.0
	github.com/WinterYukky/gorm-extra-clause-plugin v0.4.0
	github.com/dgraph-io/ristretto v0.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful-openapi/v2 v2.5.1
	github.com/emicklei/go-restful/v3 v3.13.0
	github.com/go-openapi/spec v0.20.3
	github.com/joho/godotenv v1.5.1
	github.com/lmittmann/tint v1.1.3
	github.com/mattn/go-isatty v0.0.22
	github.com/stretchr/testify v1.11.1
	github.com/tekkamanendless/gormslog v0.1.1
	github.com/tekkamanendless/httperror v1.0.1
	github.com/tekkamanendless/restapiclient v0.1.1
	github.com/threatmate/restfulwrapper v0.1.4
	github.com/threatmate/sqlite v0.1.1
	gorm.io/gorm v1.31.1
)

require (
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/net v0.0.0-20210503060351-7fd8e65b6420 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.66.10 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.41.0 // indirect
)

replace github.com/threatmate/restfulwrapper => github.com/tekkamanendless/restfulwrapper v0.1.5-0.20260423135728-9f42d58de800

//replace github.com/threatmate/restfulwrapper => ../../tekkamanendless/restfulwrapper
