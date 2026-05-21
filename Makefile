.PHONY: embedding-server go-server serve init

init:
	@echo "installing embedding-server packages" && cd embedding-server && npm install
	@echo "installing server packages" && cd server && go mod tidy
	@echo "writing VAPID keys" && cd server && go run init/main.go
	@echo "setting up Docker Compose services" && docker compose up -d

embedding-server:
	cd embedding-server && npm start

go-server:
	cd server && genkit start -- go run cmd/main.go

serve:
	@make -j 2 embedding-server go-server
