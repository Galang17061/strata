#!/bin/sh
set -e
cd "$(dirname "$0")/../.."
set -a
. ./.env
set +a
sh ops/backup.sh
LATEST=$(ls -1t "${STRATA_BACKUP_DIR:-./backups}"/strata-*.bak | head -1)
docker cp "$LATEST" strata-db:/var/opt/mssql/loadtest.bak
docker exec strata-db /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "$STRATA_DB_SA_PASSWORD" -C -Q "IF DB_ID(N'stratatest') IS NOT NULL BEGIN ALTER DATABASE stratatest SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE stratatest; END RESTORE DATABASE stratatest FROM DISK = N'/var/opt/mssql/loadtest.bak' WITH MOVE N'strata' TO N'/var/opt/mssql/data/stratatest.mdf', MOVE N'strata_log' TO N'/var/opt/mssql/data/stratatest_log.ldf', REPLACE"
docker exec strata-db rm -f /var/opt/mssql/loadtest.bak
if ! grep -q "^STRATA_TEST_DB_CONNECTION=" .env; then
  MAIN=$(grep "^STRATA_DB_CONNECTION=" .env | cut -d= -f2-)
  echo "STRATA_TEST_DB_CONNECTION=$(echo "$MAIN" | sed 's/database=strata/database=stratatest/')" >> .env
fi
echo "test database stratatest is a fresh copy; point the load at strata-api-test:5001"
