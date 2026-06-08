.PHONY: compile-api openapi-format
compile-api:
	cd tools && go tool oapi-codegen -config ../API/config.yaml ../API/api.yaml
openapi-format:
	npx openapi-format API/api.yaml -o API/api.yaml --split