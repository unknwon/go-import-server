NAME = go-import-server
LDFLAGS += -X "main.Version=$(shell git rev-parse HEAD)"

build:
	go build -v -o $(NAME)

web: build
	./$(NAME)

release:
	env GOOS=darwin GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf darwin_amd64.tar.gz $(NAME)
	env GOOS=linux GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf linux_amd64.tar.gz $(NAME)
	env GOOS=linux GOARCH=386 go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf linux_386.tar.gz $(NAME)
	env GOOS=linux GOARCH=arm go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf linux_arm.tar.gz $(NAME)
	env GOOS=windows GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf windows_amd64.tar.gz $(NAME)
	env GOOS=windows GOARCH=386 go build -ldflags '$(LDFLAGS)' -o $(NAME); tar czf windows_386.tar.gz $(NAME)

clean:
	go clean
	rm -f *.tar.gz
