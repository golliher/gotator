releases: mac-release linux-release linux-pi-release

clean:
	rm gotator-linux gotator-mac gotator-pi

mac-release:
	go build -race -o gotator-mac

# Linux releases not built with -race because requires CGO_ENABLED=1
# and I'm too lazay today to make that work
linux-release:
	env GOOS=linux go build -o gotator-linux

linux-pi-release:
	env GOOS=linux GOARCH=arm GOARM=5 go build -o gotator-pi
