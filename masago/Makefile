GO           ?= go

## dev_run: Run the application main file.
dev_run:
	$(GO) run .

## prod_run: Build and run the application.
prod_run: build
	./game

## build: Build the application.
build:
	mkdir -p bin
	rm -rf bin/masago
	$(GO) build -o bin/masago main.go

clean:
	rm -rf bin