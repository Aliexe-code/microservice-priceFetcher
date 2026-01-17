build:
	go build -o ./bin/priceFetcher ./cmd/server

run: build
	./bin/priceFetcher

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	proto/service.proto

test:
	go test -v ./...

test-coverage:
	go test -cover ./...

test-coverage-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f ./bin/priceFetcher
	rm -f coverage.out coverage.html

# Docker commands
docker-build:
	docker build -t price-fetcher:latest .

docker-run:
	docker run -p 8080:8080 -p 8081:8081 \
		-e USE_REAL_DATA=false \
		-e ALPHA_VANTAGE_API_KEY=demo \
		price-fetcher:latest

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-compose-logs:
	docker-compose logs -f

docker-clean:
	docker system prune -f

.PHONY: build run proto test test-coverage test-coverage-html clean docker-build docker-run docker-compose-up docker-compose-down docker-compose-logs docker-clean