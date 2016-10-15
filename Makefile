releases: mac-release linux-release

mac-release:
	go build -o gotator-mac

linux-release:
	env GOOS=linux go build -o gotator-linux
