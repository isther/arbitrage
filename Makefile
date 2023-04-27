build-cmd:
	go build -o cmd bin/cmd/main.go

run: clean 
	@docker-compose up -d --force-recreate --build cmd
