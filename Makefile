.PHONY: help run-api run-consumer build-api build-consumer docker-build-api docker-build-consumer test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run-api: ## Run API server locally
	go run cmd/api/main.go

run-consumer: ## Run consumer service locally
	go run cmd/consumer/main.go