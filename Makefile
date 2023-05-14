build-cmd:
	go build -o cmd bin/cmd/main.go

logs: 
	docker logs binance-mexc-cmd

docker-cmd-run:
	@docker-compose up -d --force-recreate --build cmd
