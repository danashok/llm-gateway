.PHONY: build run test clean swag db-up db-down app-up app-down

build:
	go build -o bin/main cmd/main.go

run:
	go run cmd/main.go

test:
	go test -v ./...

clean:
	rm -rf bin/

swag:
	swag init -g cmd/main.go -o openapi/ --parseDependency --parseInternal

db-up:
	docker-compose -f docker-compose/infra.yml up -d

db-down:
	docker-compose -f docker-compose/infra.yml down

app-up:
	docker-compose -f docker-compose/local.yml up -d

app-down:
	docker-compose -f docker-compose/local.yml down
