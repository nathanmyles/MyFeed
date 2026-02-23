.PHONY: daemon daemon-all ui-dev build clean

DAEMON_DIR := daemon
UI_DIR := ui
BUILD_DIR := build

daemon:
	cd $(DAEMON_DIR) && go build -o bin/myfeed-daemon .

daemon-all:
	cd $(DAEMON_DIR) && \
	GOOS=windows GOARCH=amd64 go build -o bin/myfeed-daemon-windows-amd64.exe . && \
	GOOS=darwin GOARCH=amd64 go build -o bin/myfeed-daemon-darwin-amd64 . && \
	GOOS=darwin GOARCH=arm64 go build -o bin/myfeed-daemon-darwin-arm64 . && \
	GOOS=linux GOARCH=amd64 go build -o bin/myfeed-daemon-linux-amd64 .

ui-dev: daemon
	mkdir -p $(UI_DIR)/resources/daemon
	cp $(DAEMON_DIR)/bin/myfeed-daemon $(UI_DIR)/resources/daemon/
	cd $(UI_DIR) && npm run dev

build: daemon
	mkdir -p $(UI_DIR)/resources/daemon
	cp $(DAEMON_DIR)/bin/myfeed-daemon $(UI_DIR)/resources/daemon/
	cd $(UI_DIR) && npm run build

clean:
	rm -rf $(DAEMON_DIR)/bin
	rm -rf $(UI_DIR)/resources/daemon
	rm -rf $(BUILD_DIR)

test-daemon:
	cd $(DAEMON_DIR) && go test ./...

test:
	cd $(DAEMON_DIR) && go test ./...
	cd $(UI_DIR) && npm test
