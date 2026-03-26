# Knowledge-srv 3-Layer Refactor: Input & Specifications

**Date:** 2026-03-25
**Status:** Source of Truth
**Purpose:** Define contracts for knowledge-srv refactoring and provide sample data for testing

---

## Files & Purpose

### 📋 **contract.md** — OFFICIAL CONTRACT

**Audience:** Analysis-srv team (producers)
**Authority:** Source of truth for Kafka message format
**Content:** Complete schema specification for 3 layers

- **Section 1-4:** Kafka message payload schemas
  - `analytics.batch.completed` (Layer 3: Per-document evidence)
  - `analytics.insights.published` (Layer 2: Narrative insights)
  - `analytics.report.digest` (Layer 1: Campaign overview)

- **Section 5:** Shared conventions (run_id, should_index, timestamps, Kafka config)
- **Section 6:** What knowledge-srv does NOT need

**Usage:**

- Provide to analysis-srv team as "Follow this contract exactly"
- Reference during implementation & testing
- Use for validation & error handling

---

### 📋 **master-plan.md** — IMPLEMENTATION GUIDE

**Audience:** Knowledge-srv team (consumers)
**Authority:** Architectural blueprint + detailed task breakdown
**Content:** Why migrate + how to implement

- **Section 1-3:** Context, 3-layer architecture, relationships, examples
- **Section 4:** Detailed payload contracts per layer (same as contract.md, with implementation notes)
- **Implementation notes:** What needs to change, struct definitions, Qdrant collection mapping

**Usage:**

- Reference for understanding refactor scope
- Use InsightMessage struct definition from Section 4.1
- Follow implementation checklist for each layer

---

## Sample Data — For Smoke Testing

### 📁 `uap-batches/`

**Type:** Raw crawler output (Layer 3 INPUT)
**Source:** Original social media crawl
**Used by:** Understand data flow from crawler → analysis-srv

### 📁 `outputSRM/`

**Type:** Analysis-srv enriched output (Layer 2 + Layer 1 examples)

- `insights/insights.jsonl` — Layer 2 insight cards (7 examples)
- `reports/bi_reports.json` — Layer 1 BI digest

**Used by:**

- Verify Kafka messages match contract.md
- Integration testing
- Understand actual vs expected format

---

## Key Decisions (Locked In)

### ✅ Layer 3: Direct Payload (not file reference)

- All 2000 documents embedded in single Kafka message
- Kafka `max.message.bytes ≥ 10MB` required

### ✅ Layer 2 + 3: Requires Envelope Fields

- `project_id`, `campaign_id`, `run_id` must be present
- Enables cross-layer reference & scoping

### ✅ All Layers: `should_index` Gate

- `false` → skip indexing (spam, test run, low confidence)
- `true` → index to Qdrant

### ✅ Timestamps: RFC3339 UTC Only

- Format: `YYYY-MM-DDTHH:MM:SSZ`
- No timezone offsets, no embedded strings

### ✅ Case Sensitivity

- Enums lowercase: `post`, `comment`, `reply`, `video`, `text`
- Platform enum uppercase: `TIKTOK`, `FACEBOOK`, `INSTAGRAM`
- Sentiment uppercase: `POSITIVE`, `NEGATIVE`, `NEUTRAL`, `MIXED`

---

## Implementation Sequence

### Phase 1: Contract Finalization (This phase)

✅ contract.md frozen as source of truth
✅ master-plan.md frozen as implementation guide
✅ Sample data ready for validation

### Phase 2: Analysis-srv Adjustments

- Refactor Layer 3 publisher: embed enriched documents with NLP
- Refactor Layer 2 publisher: add envelope fields
- Refactor Layer 1 publisher: flatten structure
- Validate against contract.md Section 1-4

### Phase 3: Knowledge-srv Implementation

- Update internal types (InsightMessage)
- Implement 3 Kafka consumers
- Add layer routing & Qdrant upsert logic
- Validate against master-plan.md Section 4

### Phase 4: Testing

- Smoke test with sample data (uap-batches → contract payloads)
- Integration test end-to-end
- Cross-layer reference validation

---

## Validation Checklist

### For Analysis-srv Team

Before publishing to production Kafka:

- [ ] Every `analytics.batch.completed` message matches contract.md Section 1-2
  - [ ] Has `project_id`, `campaign_id` at envelope level
  - [ ] Each document in `documents[]` is valid InsightMessage (contract.md Section 2)
  - [ ] All required fields present: identity, content, nlp, business, rag

- [ ] Every `analytics.insights.published` message matches contract.md Section 3
  - [ ] Has `project_id`, `campaign_id`, `run_id`
  - [ ] `analysis_window_start`, `analysis_window_end` are separate RFC3339 fields
  - [ ] `should_index` boolean present
  - [ ] `supporting_metrics` matches `insight_type` schema

- [ ] Every `analytics.report.digest` message matches contract.md Section 4
  - [ ] Has `project_id`, `campaign_id`, `run_id`
  - [ ] `domain_overlay`, `platform`, `total_mentions` present
  - [ ] `top_entities[]`, `top_topics[]`, `top_issues[]` are flat arrays at root level
  - [ ] `should_index` boolean present

### For Knowledge-srv Team

Before deploying consumers:

- [ ] Kafka consumers created for all 3 topics
  - [ ] `knowledge-indexing-batch` (Layer 3)
  - [ ] `knowledge-indexing-insights` (Layer 2)
  - [ ] `knowledge-indexing-digest` (Layer 1)

- [ ] Deserialization validates against contract.md
- [ ] Qdrant collections created
  - [ ] `proj_{project_id}` for Layer 3
  - [ ] `macro_insights` for Layer 1+2
- [ ] Cross-layer reference logic tested (Layer 2 evidence_references → Layer 3)
- [ ] Smoke test passes with sample data from `uap-batches/` + `outputSRM/`

---

## Questions?

- **On contract interpretation:** See contract.md Sections 1-6
- **On implementation approach:** See master-plan.md Section 4
- **On field definitions:** See contract.md field reference tables
- **On examples:** See sample data in uap-batches/, outputSRM/
