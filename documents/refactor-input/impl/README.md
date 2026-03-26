# Implementation Specs — knowledge-srv 3-Layer Migration

## Thứ tự implement

```
M1 → M2 → M3 → M4 → M5 → M6
          ↑___↑  (M3 và M4 có thể parallel)
```

## Files

| File | Milestone | Mô tả |
|------|-----------|-------|
| [M1-dynamic-collection.md](M1-dynamic-collection.md) | Foundation | Xóa hardcoded `smap_analytics`, thêm `CollectionName` vào tất cả Point domain types/options/usecases |
| [M2-type-definitions.md](M2-type-definitions.md) | Types | Kafka message structs (InsightMessage, InsightsPublishedMessage, ReportDigestMessage) + domain input/output types |
| [M3-layer2-consumer.md](M3-layer2-consumer.md) | Layer 2 | Consumer `analytics.insights.published` → `macro_insights` |
| [M4-layer1-consumer.md](M4-layer1-consumer.md) | Layer 1 | Consumer `analytics.report.digest` → prose generation → `macro_insights` |
| [M5-layer3-migration.md](M5-layer3-migration.md) | Layer 3 | **HIGH RISK** — Migrate `BatchCompletedMessage` từ MinIO → direct payload |
| [M6-collection-autocreate.md](M6-collection-autocreate.md) | Finalize | Auto-create Qdrant collections + end-to-end smoke test |

## Convention cần follow trong MỌI file

1. **Log prefix**: `{domain}.{layer}.{package}.{method}:` — e.g. `indexing.delivery.kafka.consumer.handleInsightsPublishedMessage:`
2. **Error handling trong handler**: `return nil` khi skip-worthy (bad JSON, validation fail, should_index=false) → commit offset; `return error` khi retryable (embedding fail, Qdrant fail)
3. **Naming**: `ConsumeXxx` (consumer methods), `xxxHandler` (structs), `handleXxxMessage` (workers), `toXxxInput` (presenters)
4. **Test**: `{file}_test.go` cùng package, dùng mockery mocks

## Sample data cho testing

- Layer 3 input: `documents/refactor-input/uap-batches/`
- Layer 2 sample: `documents/refactor-input/outputSRM/insights/insights.jsonl`
- Layer 1 sample: `documents/refactor-input/outputSRM/reports/bi_reports.json`
- Contract spec: `documents/refactor-input/contract.md`
