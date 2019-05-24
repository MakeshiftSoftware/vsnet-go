GO           ?= go

## dev_run: Run the application main file.
dev_run:
	$(GO) run .

## prod_run: Build and run the application.
prod_run: build
	./game

## build: Build the application.
build:
	rm -f game
	$(GO) build -o game main.go