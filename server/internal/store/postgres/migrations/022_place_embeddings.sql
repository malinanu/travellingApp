-- Semantic-search vector for Local Guide (see internal/recommend,
-- internal/http/embedding_worker.go). Stored as JSONB rather than a
-- pgvector `vector(N)` column: Postgres here is externally managed with no
-- guaranteed pgvector extension, and this avoids needing to pin the
-- embedding model's output dimension in the schema. Nullable — existing
-- places start unembedded and the background worker backfills them.
ALTER TABLE places ADD COLUMN embedding JSONB;
