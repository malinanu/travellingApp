-- Business Registration certificate: Gemini-assisted number pre-fill (see
-- internal/gemini) + Sri Lanka DRC eROC verification (see internal/eroc),
-- swept in the background by StartBRCertScheduler
-- (internal/http/br_cert_scheduler.go). See media.go's "br_cert" role
-- special-cases for how these fields get set.
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_number TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_ocr_status TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_verify_status TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_company_name TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_message TEXT NOT NULL DEFAULT '';
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS br_cert_verified_at TIMESTAMPTZ;
