build:
	go build -o w ./cmd/wheres-is-my-pizza/main.go
up:
	docker-compose up --build
down:
	docker-compose down
