-- Migration: 010 - Drop notebook/maestro tables (feature removed)
-- These tables supported the NotebookLM integration which has been stripped.

DROP TABLE IF EXISTS knowledge.notebook_chat_jobs CASCADE;
DROP TABLE IF EXISTS knowledge.notebook_sources CASCADE;
DROP TABLE IF EXISTS knowledge.notebook_campaigns CASCADE;
DROP TABLE IF EXISTS knowledge.maestro_sessions CASCADE;
