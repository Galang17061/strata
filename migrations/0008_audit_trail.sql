IF OBJECT_ID(N'dbo.AuditTrail', N'U') IS NULL
CREATE TABLE dbo.AuditTrail (
    audit_trail_id NVARCHAR(50) NOT NULL CONSTRAINT PK_AuditTrail PRIMARY KEY,
    user_name NVARCHAR(255) NOT NULL,
    method NVARCHAR(10) NOT NULL,
    path NVARCHAR(400) NOT NULL,
    status_code INT NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_AuditTrail_created_at DEFAULT GETDATE()
);
GO
