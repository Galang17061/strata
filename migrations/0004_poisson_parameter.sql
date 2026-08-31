IF OBJECT_ID(N'dbo.PoissonParameter', N'U') IS NULL
CREATE TABLE dbo.PoissonParameter (
    poisson_parameter_id NVARCHAR(50) NOT NULL CONSTRAINT PK_PoissonParameter PRIMARY KEY,
    system_component_id NVARCHAR(50) NOT NULL,
    failure_event_hours INT NOT NULL,
    n INT NOT NULL,
    rate DECIMAL(38, 18) NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_PoissonParameter_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_PoissonParameter_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_PoissonParameter_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);
GO
