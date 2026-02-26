.PHONY: help backend-run backend-test smoke quick

help:
	@echo "Available targets:"
	@echo "  make backend-run   # Run Go backend server"
	@echo "  make backend-test  # Run backend tests"
	@echo "  make smoke         # Run backend smoke checks"
	@echo "  make quick         # backend-test + smoke"

backend-run:
	cd backend && go run ./cmd/server/main.go

backend-test:
	cd backend && go test ./...

smoke:
	./scripts/smoke-test.sh

quick: backend-test smoke
