BINARY=app

run:
	@go build -o $(BINARY) ./main.go
	@./$(BINARY) --path $(path)