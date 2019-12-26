build:
	go build -o bin/redirect_default
	chmod +x bin/redirect_default
	GOOS=linux GOARCH=amd64 go build -o bin/redirect_linux_amd64
	chmod +x bin/redirect_linux_amd64