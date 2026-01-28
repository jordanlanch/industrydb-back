# Database Migrations

## How to Run Migrations

### Prerequisites
- PostgreSQL 15+
- Database created and configured

### Running Migrations

```bash
# Option 1: Using psql command
psql -h localhost -U industrydb -d industrydb -f 006_create_industries_table.sql

# Option 2: Using environment variable
export DATABASE_URL="postgres://industrydb:password@localhost:5432/industrydb?sslmode=disable"
psql $DATABASE_URL -f 006_create_industries_table.sql

# Option 3: From Docker
docker exec -i industrydb-postgres psql -U industrydb -d industrydb < 006_create_industries_table.sql
```

### Verify Migration

```bash
# Check if industries table exists
psql -h localhost -U industrydb -d industrydb -c "\dt industries"

# Check industries count (should be 20)
psql -h localhost -U industrydb -d industrydb -c "SELECT COUNT(*) FROM industries;"

# View all industries
psql -h localhost -U industrydb -d industrydb -c "SELECT id, name, category FROM industries ORDER BY sort_order;"
```

## Migration 006: Create Industries Table

**Date:** 2026-01-27
**Description:** Add industries table to support 20 industries organized in 6 categories

### What it does:
1. Creates `industries` table with all fields
2. Creates indexes for better performance
3. Adds `updated_at` trigger
4. Seeds 20 industries across 6 categories

### Industries Added:
- **Personal Care & Beauty (5):** tattoo, beauty, barber, spa, nail_salon
- **Health & Wellness (4):** gym, dentist, pharmacy, massage
- **Food & Beverage (4):** restaurant, cafe, bar, bakery
- **Automotive (3):** car_repair, car_wash, car_dealer
- **Retail (2):** clothing, convenience
- **Professional Services (2):** lawyer, accountant

### Rollback

```sql
BEGIN;
DROP TRIGGER IF EXISTS trigger_industries_updated_at ON industries;
DROP FUNCTION IF EXISTS update_industries_updated_at();
DROP TABLE IF EXISTS industries;
COMMIT;
```
