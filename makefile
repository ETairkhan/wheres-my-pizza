build:
	go build -o  where-is-my-pizza ./cmd/where-is-my-pizza/main.go
up:
	docker-compose up --build
down:
	docker-compose down
