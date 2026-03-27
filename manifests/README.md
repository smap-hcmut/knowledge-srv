# Kubernetes Manifests

Thư mục này chứa các file manifest để deploy knowledge-srv lên Kubernetes.

## Setup

### 1. Tạo file config từ example

```bash
# Copy và chỉnh sửa ConfigMap
cp manifests/configmap.yaml.example manifests/configmap.yaml

# Copy và chỉnh sửa Secret
cp manifests/secret.yaml.example manifests/secret.yaml
```

### 2. Cập nhật các giá trị trong configmap.yaml

Thay đổi các giá trị `CHANGE_ME`:
- `MINIO_ENDPOINT`: IP:port của MinIO server
- `POSTGRES_HOST`: IP của PostgreSQL server
- `QDRANT_HOST`: IP của Qdrant server

### 3. Cập nhật các secret trong secret.yaml

Thay đổi tất cả các giá trị `CHANGE_ME`:
- `POSTGRES_USER`, `POSTGRES_PASSWORD`: Credentials PostgreSQL
- `REDIS_PASSWORD`: Password Redis
- `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`: Credentials MinIO
- `VOYAGE_API_KEY`: API key từ Voyage AI
- `GEMINI_API_KEY`: API key từ Google Gemini
- `JWT_SECRET_KEY`: Random string ít nhất 32 ký tự
- `ENCRYPTER_KEY`: Random string ít nhất 32 ký tự
- `INTERNAL_KEY`: Random string cho internal API
- `DISCORD_WEBHOOK_URL`: (Optional) Discord webhook URL

**Lưu ý**: `QDRANT_API_KEY` có thể để trống nếu Qdrant không yêu cầu authentication.

### 4. Deploy

```bash
# Apply ConfigMap và Secret
kubectl apply -f manifests/configmap.yaml
kubectl apply -f manifests/secret.yaml

# Deploy các services khác
kubectl apply -f manifests/
```

## Cấu trúc

- `configmap.yaml.example`: Template cho ConfigMap (non-sensitive config)
- `secret.yaml.example`: Template cho Secret (sensitive data)
- `configmap.yaml`: File thực tế (gitignored)
- `secret.yaml`: File thực tế (gitignored)

## Bảo mật

File `configmap.yaml` và `secret.yaml` đã được thêm vào `.gitignore` để tránh commit sensitive data lên Git.

## Ports

- Qdrant gRPC: 6334 (được dùng bởi Go client)
- Qdrant REST API: 6333 (dùng cho testing/debugging)
