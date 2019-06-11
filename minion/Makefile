GO           ?= go

## dev: Run the application main file.
dev:
	$(GO) run .

## prod: Build and run the application.
prod: build
	./bin/minion

## build: Build the application.
build:
	rm -rf bin
	$(GO) build -o bin/minion main.go

clean:
	rm -rf bin