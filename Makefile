.PHONY: compile-api openapi-format openapi-bundle openapi-bundle-format ssl-generate run-prod run-dev clear-db-dev

compile-api:#generate code
	cd tools && go tool oapi-codegen -config ../api/config.yaml ../api/api.yaml && mv ./api.gen.go ../api

openapi-format:#split api
	npx openapi-format api/api.yaml -o api/api.yaml --split

openapi-bundle:#from api to bundle
	npx --yes @redocly/cli bundle api/api.yaml -o api/api.bundle.yaml

openapi-bundle-format:#from bundle to api
	npx openapi-format api/api.bundle.yaml -o api/api.yaml --split

ssl-generate:
	cd ../task-tracker-frontend/ssl && ./generate.sh
run-dev:
	docker compose up --build
run-prod:
	docker compose -f docker-compose.prod.yaml pull
	docker compose -f docker-compose.prod.yaml up
clear-db-dev:
	docker compose down -v