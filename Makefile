GO=go
BIN=winapps_systray

build:
	export PATH=$(PATH):/usr/local/go/bin && $(GO) build -o $(BIN) main.go

run: build
	./$(BIN)

clean:
	rm -f $(BIN)
