.PHONY: compile-api openapi-format
compile-api:
	 cd API && go tool oapi-codegen -config config.yaml api.yaml
openapi-format:
	npx openapi-format API/api.yaml -o API/api.yaml --split