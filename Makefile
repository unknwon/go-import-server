build:
	go build -v -o go-import-server

web: build
	./go-import-server

clean:
	go clean
