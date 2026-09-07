#!/bin/sh
set -e
STAMP=$(date +%Y%m%d-%H%M%S)
TARGET_DIR="${STRATA_BACKUP_DIR:-./backups}"
KEEP="${STRATA_BACKUP_KEEP:-14}"
mkdir -p "$TARGET_DIR"
docker exec strata-db pg_dump -U postgres -Fc strata > "$TARGET_DIR/strata-$STAMP.dump"
ls -1t "$TARGET_DIR"/strata-*.dump | tail -n +$((KEEP + 1)) | xargs -r rm -f
echo "backup written: $TARGET_DIR/strata-$STAMP.dump"
echo "remember: copy this file OFF this machine (object storage, another host) or it protects nothing"
