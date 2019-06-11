GO           ?= go

## dev: Run the application main file.
dev:
	$(GO) run .

## prod: Build and run the application.
prod: build
	./bin/master

## build: Build the application.
build:
	rm -rf bin
	$(GO) build -o bin/master main.go

clean:
	rm -rf bin