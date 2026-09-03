IF OBJECT_ID(N'dbo.SystemThreshold', N'U') IS NULL
CREATE TABLE dbo.SystemThreshold (
    rbd_system_id NVARCHAR(50) NOT NULL CONSTRAINT PK_SystemThreshold PRIMARY KEY,
    threshold DECIMAL(38, 18) NOT NULL,
    updated_by NVARCHAR(100) NULL,
    updated_at DATETIME NOT NULL CONSTRAINT DF_SystemThreshold_updated_at DEFAULT GETDATE(),
    CONSTRAINT FK_SystemThreshold_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);
GO
IF OBJECT_ID(N'dbo.Notification', N'U') IS NULL
CREATE TABLE dbo.Notification (
    notification_id NVARCHAR(50) NOT NULL CONSTRAINT PK_Notification PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NOT NULL,
    title NVARCHAR(200) NOT NULL,
    body NVARCHAR(400) NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_Notification_created_at DEFAULT GETDATE()
);
GO
