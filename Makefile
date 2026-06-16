.PHONY: compile-api openapi-format openapi-bundle openapi-bundle-format

compile-api:
	cd tools && go tool oapi-codegen -config ../api/config.yaml ../api/api.yaml && mv ./api.gen.go ../api

openapi-format:
	npx openapi-format api/api.yaml -o api/api.yaml --split

openapi-bundle:
	npx openapi-format api/api.yaml -o api/api.bundle.yaml

openapi-bundle-format:
	npx openapi-format api/api.bundle.yaml -o api/api.yaml --split