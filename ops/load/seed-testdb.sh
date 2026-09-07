#!/bin/sh
set -e
cd "$(dirname "$0")/../.."
set -a
. ./.env
set +a
sh ops/backup.sh
LATEST=$(ls -1t "${STRATA_BACKUP_DIR:-./backups}"/strata-*.dump | head -1)
docker cp "$LATEST" strata-db:/tmp/loadtest.dump
docker exec strata-db psql -U postgres -c "DROP DATABASE IF EXISTS stratatest WITH (FORCE);"
docker exec strata-db psql -U postgres -c "CREATE DATABASE stratatest;"
docker exec strata-db pg_restore -U postgres -d stratatest /tmp/loadtest.dump
docker exec strata-db rm -f /tmp/loadtest.dump
if ! grep -q "^STRATA_TEST_DB_CONNECTION=" .env; then
  MAIN=$(grep "^STRATA_DB_CONNECTION=" .env | cut -d= -f2-)
  echo "STRATA_TEST_DB_CONNECTION=$(echo "$MAIN" | sed 's#/strata?#/stratatest?#')" >> .env
fi
echo "test database stratatest is a fresh copy; point the load at strata-api-test:5001"
