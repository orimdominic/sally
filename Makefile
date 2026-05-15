.PHONY: embedding-server go-server serve

embedding-server:
	cd embedding-server && npm start

go-server:
	cd server && go run cmd/main.go

serve: 
	@make -j 2 embedding-server go-server
