test: lint unit_tests

lint:
	@if ! command -v golangci-lint; then \
		echo "linting uses golangci-lint: you can install it with:\n"; \
		echo "    brew install golangci-lint\n"; \
		exit 1; \
	fi
	golangci-lint run

unit_tests:
	go test -v ./...

update_deps:
	go get -t -u ./... && go mod tidy && go mod vendor
