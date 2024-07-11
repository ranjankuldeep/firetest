build: 
	go build -o bin/firetest
run: build
	sudo ./bin/firetest
test: 
	go test ./...