# Spotto Production Database Backup & Migration Safety Guide

This document outlines the backup strategy, disaster recovery procedures, and zero-downtime database migration guidelines for the Spotto PostgreSQL infrastructure.

---

## 1. Backup Strategy & Cadence

### Automated Daily Snapshot (WAL-G / pg_dump)
- **Schedule**: Daily at 02:00 UTC (low-traffic window for Sri Lanka timezone UTC+5:30).
- **Retention Policy**:
  - Daily backups: Retained for **30 days**.
  - Weekly snapshots: Retained for **12 weeks**.
  - Monthly snapshots: Retained for **1 year**.

### Continuous Point-In-Time Recovery (PITR)
- **WAL Archiving**: PostgreSQL Write-Ahead Logs (`WAL`) are streamed to S3 / Cloudflare R2 object storage every 5 minutes.
- **RPO (Recovery Point Objective)**: Max 5 minutes data loss.
- **RTO (Recovery Time Objective)**: < 30 minutes for full DB restore.

---

## 2. Automated Backup Execution Commands

### Manual Backup (pg_dump)
To perform an immediate compressed snapshot of the production database:

```bash
docker exec -t spotto_postgres pg_dump \
  -U spotto \
  -d spotto \
  -F c \
  -b \
  -v \
  -f /backups/spotto_prod_$(date +%Y%m%d_%H%M%S).dump
```

### Restore Procedure
To restore a backup into a new or recovered PostgreSQL instance:

```bash
# 1. Create target database
docker exec -i spotto_postgres createdb -U spotto spotto_recovery

# 2. Restore schema and data
docker exec -i spotto_postgres pg_restore \
  -U spotto \
  -d spotto_recovery \
  -v /backups/spotto_prod_20260804_020000.dump
```

---

## 3. Migration Safety Guidelines

### Schema Migration Execution
Migrations are executed via `server/internal/store/postgres/migrations/` on server startup inside an explicit transaction block:

1. **Transactional Migrations**: Always wrap schema alterations in `BEGIN;` ... `COMMIT;`.
2. **Backward-Compatible Alterations**:
   - Adding columns: Always specify `NULL` or provide a non-locking default (`DEFAULT ...`).
   - Renaming columns: Use a two-phase process (add new column -> dual write -> backfill -> drop old column).
   - Creating indexes: Use `CREATE INDEX CONCURRENTLY` in production to prevent table lockups during high traffic.
3. **Rollback Verification**: Every migration file MUST have a corresponding rollback test in `store_test.go`.
