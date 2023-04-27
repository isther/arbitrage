build-cmd:
	go build -o cmd bin/cmd/main.go

docker-cmd-run:
	@docker-compose up -d --force-recreate --build cmd
