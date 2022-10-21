docker-compose-up:
	docker-compose up

docker-compose-down:
	docker-compose down

test:
	go fmt ./...
	go test -cover ./...