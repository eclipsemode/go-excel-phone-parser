BINARY=app

run:
	@go build -o $(BINARY) ./main.go
	@./$(BINARY) --path $(path) --city $(city) --from $(from) --to $(to)