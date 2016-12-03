releases: mac-release linux-release

clean:
	rm gotator-linux gotator-mac

mac-release:
	go build -race -o gotator-mac

# Linux releases not built with -race because requires CGO_ENABLED=1
# and I'm too lazay today to make that work
linux-release:
	env GOOS=linux go build -o gotator-linux
