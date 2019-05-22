GO           ?= go
GOFMT        ?= $(GO)fmt
pkgs          = ./...

## install_revive: Install revive for linting.
install_revive:
	@echo ">> Install revive"
	$(GO) get github.com/mgechev/revive

## style: Check code style.
style:
	@echo ">> checking code style"
	@fmtRes=$$($(GOFMT) -d $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	if [ -n "$${fmtRes}" ]; then \
		echo "gofmt checking failed!"; echo "$${fmtRes}"; echo; \
		echo "Please ensure you are using $$($(GO) version) for formatting code."; \
		exit 1; \
	fi

## lint: Lint the code.
lint:
	@echo ">> Lint all files"
	revive -config revive-config.toml -exclude vendor/... -formatter friendly ./...

## format: Format the code.
format:
	@echo ">> formatting code"
	$(GO) fmt $(pkgs)

## vet: Examines source code and reports suspicious constructs.
vet:
	@echo ">> vetting code"
	$(GO) vet $(pkgs)

## test_short: Run test cases with short flag.
test_short:
	@echo ">> running short tests"
	$(GO) test -short $(pkgs)

## test: Run test cases.
test:
	@echo ">> running all tests"
	$(GO) test -race -cover $(pkgs)

## coverage: Create HTML coverage report
coverage:
	rm -f coverage.html cover.out
	$(GO) test -coverprofile=cover.out $(pkgs)
	go tool cover -html=cover.out -o coverage.html

## ci: Run all CI tests.
ci: style test vet lint
	@echo "\n==> All quality checks passed"

help: Makefile
	@echo
	@echo " Choose a command run in Vsnet:"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo