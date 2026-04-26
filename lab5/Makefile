BIN := go2web

.PHONY: build run clean install

build:
	go build -o $(BIN) .

run: build
	./$(BIN) $(ARGS)

clean:
	rm -f $(BIN)

install: build
	cp $(BIN) $(GOPATH)/bin/$(BIN)
