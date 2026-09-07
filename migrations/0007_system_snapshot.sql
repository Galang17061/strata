CREATE TABLE IF NOT EXISTS dbo.SystemSnapshot (
    system_snapshot_id varchar(50) NOT NULL CONSTRAINT PK_SystemSnapshot PRIMARY KEY,
    rbd_system_id varchar(50) NOT NULL,
    label varchar(200) NOT NULL,
    kind varchar(20) NOT NULL,
    payload text NOT NULL,
    created_by varchar(100) NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT FK_SystemSnapshot_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);
