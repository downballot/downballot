.PHONY: all
all:

.PHONY: test
test:
	go test ./...

SWAGGER_UI_VERSION := v5.32.4
SWAGGER_UI_ZIP := bin/swagger-ui-$(SWAGGER_UI_VERSION).zip
SWAGGER_UI_URL := https://github.com/swagger-api/swagger-ui/archive/refs/tags/$(SWAGGER_UI_VERSION).zip

bin:
	mkdir -p bin

.PHONY: swagger
swagger: bin
	@echo "Downloading Swagger UI..."
	curl -L $(SWAGGER_UI_URL) -o $(SWAGGER_UI_ZIP)
	@echo "Extracting Swagger UI..."
	rm -rf static/swagger
	unzip -j $(SWAGGER_UI_ZIP) "*/dist/*" -d static/swagger
	rm $(SWAGGER_UI_ZIP)
	@echo "Replacing petstore reference..."
	for file in $$(grep -rl petstore static/swagger/); do \
		sed -i "$$file" -e 's_https://petstore.swagger.io/v2/swagger.json_/api/swagger.json_g'; \
	done
	@echo "Swagger UI installed in static/swagger"

.PHONY: run
run:
	go generate ./...
	go run ./cmd/web/...

