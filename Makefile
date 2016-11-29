releases: mac-release linux-release

clean:
	rm gotator-linux gotator-mac

mac-release:
	go build -o gotator-mac

linux-release:
	env GOOS=linux go build -o gotator-linux
