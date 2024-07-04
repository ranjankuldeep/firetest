build: 
	@go build -o bin/firetest
run: build
	@sudo -E ./bin/firetest
test: 
	@go test ./...