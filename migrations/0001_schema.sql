IF OBJECT_ID(N'dbo.Roles', N'U') IS NULL
CREATE TABLE dbo.Roles (
    Id UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_Roles PRIMARY KEY,
    RoleName NVARCHAR(255) NOT NULL
);
GO

IF OBJECT_ID(N'dbo.Users', N'U') IS NULL
CREATE TABLE dbo.Users (
    Id UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_Users PRIMARY KEY,
    Fullname NVARCHAR(255) NOT NULL,
    UserName NVARCHAR(255) NOT NULL,
    Email NVARCHAR(255) NOT NULL,
    Password NVARCHAR(MAX) NOT NULL
);
GO

IF OBJECT_ID(N'dbo.UserRole', N'U') IS NULL
CREATE TABLE dbo.UserRole (
    UserId UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_UserRole PRIMARY KEY,
    RoleId UNIQUEIDENTIFIER NOT NULL,
    CONSTRAINT FK_UserRole_Users_UserId FOREIGN KEY (UserId) REFERENCES dbo.Users (Id) ON DELETE CASCADE,
    CONSTRAINT FK_UserRole_Roles_RoleId FOREIGN KEY (RoleId) REFERENCES dbo.Roles (Id)
);
GO

IF OBJECT_ID(N'dbo.UserAccess', N'U') IS NULL
CREATE TABLE dbo.UserAccess (
    Id UNIQUEIDENTIFIER NOT NULL CONSTRAINT PK_UserAccess PRIMARY KEY,
    UserId UNIQUEIDENTIFIER NOT NULL,
    Modul NVARCHAR(255) NOT NULL,
    is_add BIT NOT NULL CONSTRAINT DF_UserAccess_is_add DEFAULT 0,
    is_edit BIT NOT NULL CONSTRAINT DF_UserAccess_is_edit DEFAULT 0,
    is_delete BIT NOT NULL CONSTRAINT DF_UserAccess_is_delete DEFAULT 0,
    is_view BIT NOT NULL CONSTRAINT DF_UserAccess_is_view DEFAULT 0,
    is_download BIT NOT NULL CONSTRAINT DF_UserAccess_is_download DEFAULT 0,
    CONSTRAINT FK_UserAccess_Users_UserId FOREIGN KEY (UserId) REFERENCES dbo.Users (Id) ON DELETE CASCADE
);
GO

IF OBJECT_ID(N'dbo.MasterManufacturer', N'U') IS NULL
CREATE TABLE dbo.MasterManufacturer (
    vendor_id NVARCHAR(50) NOT NULL CONSTRAINT PK_MasterManufacturer PRIMARY KEY,
    manufacturer_name NVARCHAR(255) NOT NULL,
    logo_image NVARCHAR(MAX) NULL,
    valid_until DATETIME NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_MasterManufacturer_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_MasterManufacturer_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL
);
GO

IF OBJECT_ID(N'dbo.MasterComponent', N'U') IS NULL
CREATE TABLE dbo.MasterComponent (
    component_id NVARCHAR(50) NOT NULL CONSTRAINT PK_MasterComponent PRIMARY KEY,
    component_name NVARCHAR(255) NOT NULL,
    vendor_id NVARCHAR(50) NOT NULL,
    failure_rate DECIMAL(38, 18) NULL,
    cost NVARCHAR(MAX) NULL,
    compatibility NVARCHAR(MAX) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_MasterComponent_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_MasterComponent_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    serial_number NVARCHAR(255) NULL,
    CONSTRAINT FK_MasterComponent_MasterManufacturer_vendor_id FOREIGN KEY (vendor_id) REFERENCES dbo.MasterManufacturer (vendor_id)
);
GO

IF OBJECT_ID(N'dbo.MasterProject', N'U') IS NULL
CREATE TABLE dbo.MasterProject (
    project_id NVARCHAR(50) NOT NULL CONSTRAINT PK_MasterProject PRIMARY KEY,
    project_name NVARCHAR(255) NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_MasterProject_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_MasterProject_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL
);
GO

IF OBJECT_ID(N'dbo.RbdSystemDrawing', N'U') IS NULL
CREATE TABLE dbo.RbdSystemDrawing (
    rbd_system_id NVARCHAR(50) NOT NULL CONSTRAINT PK_RbdSystemDrawing PRIMARY KEY,
    project_id NVARCHAR(50) NOT NULL,
    drawing_name NVARCHAR(255) NULL,
    system_name NVARCHAR(255) NULL,
    running_hours NVARCHAR(255) NULL,
    reliability_total DECIMAL(38, 18) NULL,
    formula NVARCHAR(MAX) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_RbdSystemDrawing_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_RbdSystemDrawing_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_RbdSystemDrawing_MasterProject_project_id FOREIGN KEY (project_id) REFERENCES dbo.MasterProject (project_id) ON DELETE CASCADE
);
GO

IF OBJECT_ID(N'dbo.SystemComponentProperties', N'U') IS NULL
CREATE TABLE dbo.SystemComponentProperties (
    system_component_id NVARCHAR(50) NOT NULL CONSTRAINT PK_SystemComponentProperties PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NULL,
    parent_id NVARCHAR(20) NOT NULL CONSTRAINT DF_SystemComponentProperties_parent_id DEFAULT '',
    component_name NVARCHAR(255) NOT NULL,
    component_tag_number NVARCHAR(255) NULL,
    active INT NULL,
    vendor NVARCHAR(255) NULL,
    formula_code NVARCHAR(20) NULL,
    distribution_type NVARCHAR(50) NULL,
    failure_rate DECIMAL(38, 18) NULL,
    running_hours DECIMAL(38, 18) NULL,
    scale_parameter DECIMAL(38, 18) NULL,
    shape_parameter DECIMAL(38, 18) NULL,
    connection_type NVARCHAR(50) NULL,
    connection_to_id NVARCHAR(255) NULL,
    position_x NVARCHAR(255) NULL,
    position_y NVARCHAR(255) NULL,
    source_position NVARCHAR(50) NULL,
    target_position NVARCHAR(50) NULL,
    id_node NVARCHAR(255) NULL,
    reliability_value DECIMAL(38, 18) NULL,
    active_component INT NULL,
    total_component INT NULL,
    regresi DECIMAL(38, 18) NULL,
    mtbf DECIMAL(38, 18) NULL,
    created_at DATETIME NULL CONSTRAINT DF_SystemComponentProperties_created_at DEFAULT GETDATE(),
    updated_at DATETIME NULL CONSTRAINT DF_SystemComponentProperties_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    min_unit INT NULL CONSTRAINT DF_SystemComponentProperties_min_unit DEFAULT 1,
    max_unit INT NULL CONSTRAINT DF_SystemComponentProperties_max_unit DEFAULT 5
);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_SystemComponentProperties_connection_to_id' AND object_id = OBJECT_ID(N'dbo.SystemComponentProperties'))
CREATE UNIQUE NONCLUSTERED INDEX IX_SystemComponentProperties_connection_to_id ON dbo.SystemComponentProperties (connection_to_id) WHERE connection_to_id IS NOT NULL;
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_SystemComponentProperties_parent_id' AND object_id = OBJECT_ID(N'dbo.SystemComponentProperties'))
CREATE NONCLUSTERED INDEX IX_SystemComponentProperties_parent_id ON dbo.SystemComponentProperties (parent_id);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_SystemComponentProperties_rbd_system_id' AND object_id = OBJECT_ID(N'dbo.SystemComponentProperties'))
CREATE NONCLUSTERED INDEX IX_SystemComponentProperties_rbd_system_id ON dbo.SystemComponentProperties (rbd_system_id);
GO

IF OBJECT_ID(N'dbo.SystemComponentDrawing', N'U') IS NULL
CREATE TABLE dbo.SystemComponentDrawing (
    id_edge NVARCHAR(255) NOT NULL CONSTRAINT PK_SystemComponentDrawing PRIMARY KEY,
    source_id NVARCHAR(255) NULL,
    target_id NVARCHAR(255) NULL
);
GO

IF OBJECT_ID(N'dbo.FailureEventHistory', N'U') IS NULL
CREATE TABLE dbo.FailureEventHistory (
    failure_event_id NVARCHAR(50) NOT NULL CONSTRAINT PK_FailureEventHistory PRIMARY KEY,
    system_component_id NVARCHAR(50) NOT NULL,
    failure_date DATETIME NOT NULL,
    failure_number INT NULL,
    running_hours INT NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_FailureEventHistory_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_FailureEventHistory_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_FailureEventHistory_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);
GO

IF OBJECT_ID(N'dbo.WeibullParameter', N'U') IS NULL
CREATE TABLE dbo.WeibullParameter (
    weibull_parameter_id NVARCHAR(50) NOT NULL CONSTRAINT PK_WeibullParameter PRIMARY KEY,
    system_component_id NVARCHAR(50) NOT NULL,
    failure_event_hours INT NOT NULL,
    n INT NOT NULL,
    freq_f DECIMAL(38, 18) NOT NULL,
    x DECIMAL(38, 18) NOT NULL,
    y DECIMAL(38, 18) NOT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_WeibullParameter_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_WeibullParameter_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_WeibullParameter_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);
GO

IF OBJECT_ID(N'dbo.ExponentialParameter', N'U') IS NULL
CREATE TABLE dbo.ExponentialParameter (
    exponential_parameter_id NVARCHAR(50) NOT NULL CONSTRAINT PK_ExponentialParameter PRIMARY KEY,
    system_component_id NVARCHAR(50) NOT NULL,
    failure_event_hours INT NOT NULL,
    n INT NOT NULL,
    fregf DECIMAL(38, 18) NOT NULL,
    f_t_median_rank DECIMAL(38, 18) NOT NULL,
    r_t DECIMAL(38, 18) NOT NULL,
    inRt INT NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_ExponentialParameter_created_at DEFAULT GETDATE(),
    updated_at DATETIME NOT NULL CONSTRAINT DF_ExponentialParameter_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL
);
GO

IF OBJECT_ID(N'dbo.ReliabilityPlotComponent', N'U') IS NULL
CREATE TABLE dbo.ReliabilityPlotComponent (
    reliability_plot_id NVARCHAR(50) NOT NULL CONSTRAINT PK_ReliabilityPlotComponent PRIMARY KEY,
    system_component_id NVARCHAR(50) NOT NULL,
    time_t INT NOT NULL,
    reliability_comp DECIMAL(38, 18) NULL,
    created_at DATETIME NULL CONSTRAINT DF_ReliabilityPlotComponent_created_at DEFAULT GETDATE(),
    updated_at DATETIME NULL CONSTRAINT DF_ReliabilityPlotComponent_updated_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL,
    updated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_ReliabilityPlotComponent_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);
GO

IF OBJECT_ID(N'dbo.Hierarchy', N'U') IS NULL
CREATE TABLE dbo.Hierarchy (
    hierarchy_id NVARCHAR(20) NOT NULL CONSTRAINT PK_Hierarchy PRIMARY KEY CLUSTERED,
    rbd_system_id NVARCHAR(50) NULL,
    parent_id NVARCHAR(20) NOT NULL,
    level INT NOT NULL,
    sub_system_name NVARCHAR(255) NULL,
    formula NVARCHAR(MAX) NULL,
    formula_code NVARCHAR(20) NULL,
    connection_type NVARCHAR(255) NULL,
    realibility_value DECIMAL(38, 18) NULL,
    running_hours INT NULL,
    position_x DECIMAL(38, 18) NULL,
    position_y DECIMAL(38, 18) NULL,
    source_id NVARCHAR(255) NULL,
    target_id NVARCHAR(255) NULL,
    CONSTRAINT FK_Hierarchy_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_Hierarchy_rbd_system_id' AND object_id = OBJECT_ID(N'dbo.Hierarchy'))
CREATE NONCLUSTERED INDEX IX_Hierarchy_rbd_system_id ON dbo.Hierarchy (rbd_system_id);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_Hierarchy_parent_id' AND object_id = OBJECT_ID(N'dbo.Hierarchy'))
CREATE NONCLUSTERED INDEX IX_Hierarchy_parent_id ON dbo.Hierarchy (parent_id);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_Hierarchy_level' AND object_id = OBJECT_ID(N'dbo.Hierarchy'))
CREATE NONCLUSTERED INDEX IX_Hierarchy_level ON dbo.Hierarchy (level);
GO

IF OBJECT_ID(N'dbo.ReliabilityHistory', N'U') IS NULL
CREATE TABLE dbo.ReliabilityHistory (
    history_id NVARCHAR(50) NOT NULL CONSTRAINT PK_ReliabilityHistory PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NOT NULL,
    hierarchy_id NVARCHAR(20) NOT NULL,
    hierarchy_name NVARCHAR(255) NULL,
    hierarchy_level INT NULL,
    formula_code NVARCHAR(20) NULL,
    formula NVARCHAR(MAX) NULL,
    calculated_reliability DECIMAL(38, 18) NULL,
    reliability_lookup NVARCHAR(MAX) NULL,
    component_details NVARCHAR(MAX) NULL,
    running_hours DECIMAL(38, 18) NULL,
    calculation_timestamp DATETIME NOT NULL CONSTRAINT DF_ReliabilityHistory_calculation_timestamp DEFAULT GETDATE(),
    calculated_by NVARCHAR(100) NULL,
    CONSTRAINT FK_ReliabilityHistory_RbdSystem FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE,
    CONSTRAINT FK_ReliabilityHistory_Hierarchy FOREIGN KEY (hierarchy_id) REFERENCES dbo.Hierarchy (hierarchy_id)
);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_ReliabilityHistory_HierarchyId' AND object_id = OBJECT_ID(N'dbo.ReliabilityHistory'))
CREATE NONCLUSTERED INDEX IX_ReliabilityHistory_HierarchyId ON dbo.ReliabilityHistory (hierarchy_id, calculation_timestamp DESC);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_ReliabilityHistory_RbdSystemHierarchy' AND object_id = OBJECT_ID(N'dbo.ReliabilityHistory'))
CREATE NONCLUSTERED INDEX IX_ReliabilityHistory_RbdSystemHierarchy ON dbo.ReliabilityHistory (rbd_system_id, hierarchy_id, calculation_timestamp DESC);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_ReliabilityHistory_Timestamp' AND object_id = OBJECT_ID(N'dbo.ReliabilityHistory'))
CREATE NONCLUSTERED INDEX IX_ReliabilityHistory_Timestamp ON dbo.ReliabilityHistory (calculation_timestamp DESC);
GO

IF OBJECT_ID(N'dbo.OptimizationRun', N'U') IS NULL
CREATE TABLE dbo.OptimizationRun (
    optimization_run_id NVARCHAR(50) NOT NULL CONSTRAINT PK_OptimizationRun PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NOT NULL,
    optimization_type INT NOT NULL,
    target_reliability DECIMAL(18, 10) NULL,
    max_budget DECIMAL(18, 2) NULL,
    weight_cost DECIMAL(5, 4) NULL,
    weight_reliability DECIMAL(5, 4) NULL,
    running_hours DECIMAL(18, 4) NOT NULL CONSTRAINT DF_OptimizationRun_running_hours DEFAULT 1000,
    population_size INT NOT NULL CONSTRAINT DF_OptimizationRun_population_size DEFAULT 50,
    max_generations INT NOT NULL CONSTRAINT DF_OptimizationRun_max_generations DEFAULT 100,
    crossover_probability DECIMAL(5, 4) NOT NULL CONSTRAINT DF_OptimizationRun_crossover_probability DEFAULT 0.8,
    mutation_probability DECIMAL(5, 4) NOT NULL CONSTRAINT DF_OptimizationRun_mutation_probability DEFAULT 0.1,
    status NVARCHAR(20) NOT NULL CONSTRAINT DF_OptimizationRun_status DEFAULT 'Running',
    best_fitness DECIMAL(18, 10) NULL,
    best_reliability DECIMAL(18, 10) NULL,
    best_cost DECIMAL(18, 2) NULL,
    generations_completed INT NULL,
    execution_time_ms BIGINT NULL,
    error_message NVARCHAR(MAX) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_OptimizationRun_created_at DEFAULT GETDATE(),
    completed_at DATETIME NULL,
    created_by NVARCHAR(100) NULL,
    CONSTRAINT FK_OptimizationRun_RbdSystem FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id)
);
GO

IF OBJECT_ID(N'dbo.OptimizationResult', N'U') IS NULL
CREATE TABLE dbo.OptimizationResult (
    optimization_result_id NVARCHAR(50) NOT NULL CONSTRAINT PK_OptimizationResult PRIMARY KEY,
    optimization_run_id NVARCHAR(50) NOT NULL,
    system_component_id NVARCHAR(50) NOT NULL,
    component_name NVARCHAR(200) NOT NULL,
    original_vendor_id NVARCHAR(50) NULL,
    optimized_vendor_id NVARCHAR(50) NOT NULL,
    optimized_vendor_name NVARCHAR(200) NULL,
    original_unit_count INT NULL,
    optimized_unit_count INT NOT NULL CONSTRAINT DF_OptimizationResult_optimized_unit_count DEFAULT 1,
    connection_type NVARCHAR(50) NULL,
    failure_rate DECIMAL(18, 10) NULL,
    cost_per_unit DECIMAL(18, 2) NULL,
    total_cost DECIMAL(18, 2) NULL,
    component_reliability DECIMAL(18, 10) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_OptimizationResult_created_at DEFAULT GETDATE(),
    CONSTRAINT FK_OptimizationResult_Run FOREIGN KEY (optimization_run_id) REFERENCES dbo.OptimizationRun (optimization_run_id)
);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_OptimizationRun_RbdSystemId' AND object_id = OBJECT_ID(N'dbo.OptimizationRun'))
CREATE NONCLUSTERED INDEX IX_OptimizationRun_RbdSystemId ON dbo.OptimizationRun (rbd_system_id);
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'IX_OptimizationResult_RunId' AND object_id = OBJECT_ID(N'dbo.OptimizationResult'))
CREATE NONCLUSTERED INDEX IX_OptimizationResult_RunId ON dbo.OptimizationResult (optimization_run_id);
GO
