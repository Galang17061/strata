#!/bin/sh
set -e
STAMP=$(date +%Y%m%d-%H%M%S)
TARGET_DIR="${STRATA_BACKUP_DIR:-./backups}"
KEEP="${STRATA_BACKUP_KEEP:-14}"
PASSWORD="${STRATA_DB_SA_PASSWORD:?set STRATA_DB_SA_PASSWORD or source .env first}"
mkdir -p "$TARGET_DIR"
docker exec strata-db /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "$PASSWORD" -C -Q "BACKUP DATABASE strata TO DISK = N'/var/opt/mssql/strata-$STAMP.bak' WITH INIT, COMPRESSION"
docker cp "strata-db:/var/opt/mssql/strata-$STAMP.bak" "$TARGET_DIR/strata-$STAMP.bak"
docker exec strata-db rm -f "/var/opt/mssql/strata-$STAMP.bak"
ls -1t "$TARGET_DIR"/strata-*.bak | tail -n +$((KEEP + 1)) | xargs -r rm -f
echo "backup written: $TARGET_DIR/strata-$STAMP.bak"
echo "remember: copy this file OFF this machine (object storage, another host) or it protects nothing"
