CREATE TABLE IF NOT EXISTS dbo.SystemThreshold (
    rbd_system_id varchar(50) NOT NULL CONSTRAINT PK_SystemThreshold PRIMARY KEY,
    threshold numeric(38, 18) NOT NULL,
    updated_by varchar(100) NULL,
    updated_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT FK_SystemThreshold_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.Notification (
    notification_id varchar(50) NOT NULL CONSTRAINT PK_Notification PRIMARY KEY,
    rbd_system_id varchar(50) NOT NULL,
    title varchar(200) NOT NULL,
    body varchar(400) NOT NULL,
    created_at timestamp NOT NULL DEFAULT now()
);
