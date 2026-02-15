swagger:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go

run-api:
# 	@echo "Generating swagger"
# 	@swag init -g cmd/api/main.go
# 	@sed -i '' '/LeftDelim:/d' docs/docs.go
# 	@sed -i '' '/RightDelim:/d' docs/docs.go
# 	@echo "Running the application"
	@go run cmd/api/main.go

.PHONY: swagger run-api
