CREATE TABLE IF NOT EXISTS dbo.Job (
    job_id varchar(50) NOT NULL CONSTRAINT PK_Job PRIMARY KEY,
    kind varchar(40) NOT NULL,
    rbd_system_id varchar(50) NULL,
    status varchar(20) NOT NULL DEFAULT 'queued',
    request text NOT NULL,
    result text NULL,
    error_message text NULL,
    created_by varchar(100) NULL,
    created_by_id varchar(50) NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    started_at timestamp NULL,
    finished_at timestamp NULL,
    CONSTRAINT FK_Job_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);

ALTER TABLE dbo.Job ADD COLUMN IF NOT EXISTS created_by_id varchar(50) NULL;

CREATE INDEX IF NOT EXISTS IX_Job_waiting ON dbo.Job (status, created_at);

CREATE INDEX IF NOT EXISTS IX_Job_owner ON dbo.Job (created_by_id, created_at DESC);

CREATE INDEX IF NOT EXISTS IX_Job_system ON dbo.Job (rbd_system_id, created_at DESC);
