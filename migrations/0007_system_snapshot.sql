IF OBJECT_ID(N'dbo.SystemSnapshot', N'U') IS NULL
CREATE TABLE dbo.SystemSnapshot (
    system_snapshot_id NVARCHAR(50) NOT NULL CONSTRAINT PK_SystemSnapshot PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NOT NULL,
    label NVARCHAR(200) NOT NULL,
    kind NVARCHAR(20) NOT NULL,
    payload NVARCHAR(MAX) NOT NULL,
    created_by NVARCHAR(100) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_SystemSnapshot_created_at DEFAULT GETDATE(),
    CONSTRAINT FK_SystemSnapshot_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);
GO
