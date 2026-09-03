# Restoring Strata from a backup

This runbook brings a dead or corrupted deployment back from the latest `.bak` file.
Practise it before you need it: a backup that has never been restored is a hope, not a plan.

## What you need

- The most recent `strata-*.bak` file (from `ops/backup.sh`, fetched back from wherever
  you keep the off-machine copies).
- The `.env` of the deployment (or at least `STRATA_DB_SA_PASSWORD` and the JWT and
  password-cipher keys — without `STRATA_PASSWORD_KEY` the stored passwords are useless).
- Docker on the target machine.

## Steps

1. Start a fresh database container (skip if `strata-db` is already up and healthy):

   ```sh
   docker compose up -d strata-db
   ```

2. Copy the backup into the container:

   ```sh
   docker cp ./strata-20260903-020000.bak strata-db:/var/opt/mssql/restore.bak
   ```

3. Restore over the `strata` database. The service must not be writing while this runs,
   so stop it first:

   ```sh
   docker compose stop strata-api
   docker exec strata-db /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "$STRATA_DB_SA_PASSWORD" -C -Q "ALTER DATABASE strata SET SINGLE_USER WITH ROLLBACK IMMEDIATE; RESTORE DATABASE strata FROM DISK = N'/var/opt/mssql/restore.bak' WITH REPLACE; ALTER DATABASE strata SET MULTI_USER"
   ```

4. Start the service again (its entrypoint reapplies migrations, which are idempotent):

   ```sh
   docker compose up -d strata-api
   ```

5. Verify before declaring victory:

   - `curl http://localhost:5000/health` answers 200.
   - Sign in to the web app and open a project you know; spot-check one reliability
     figure against the number you remember.
   - `docker exec strata-db ... -Q "SELECT COUNT(*) FROM strata.dbo.MasterProject"`
     returns a believable count.

6. Clean up:

   ```sh
   docker exec strata-db rm /var/opt/mssql/restore.bak
   ```

## Scheduling backups

Run `ops/backup.sh` from cron on the host, daily, outside working hours:

```
0 2 * * * cd /path/to/strata && . ./.env && sh ops/backup.sh >> /var/log/strata-backup.log 2>&1
```

Then copy the newest file off the machine (rclone, scp, object storage — anything that
does not live on the same disk). `STRATA_BACKUP_DIR` and `STRATA_BACKUP_KEEP` control
where the files land and how many stay.

## The promise this buys

Daily backups mean at most 24 hours of work can be lost. If that is too much, raise the
cron frequency; the script is safe to run as often as you like.
