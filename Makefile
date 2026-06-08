.PHONY: compile-api openapi-format
compile-api:
	cd tools && go tool oapi-codegen -config ../api/config.yaml ../api/api.yaml && mv ./api.gen.go ../api
openapi-format:
	npx openapi-format api/api.yaml -o api/api.yaml --split