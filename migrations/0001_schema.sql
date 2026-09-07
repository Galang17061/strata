CREATE SCHEMA IF NOT EXISTS dbo;

CREATE TABLE IF NOT EXISTS dbo.Roles (
    Id uuid NOT NULL CONSTRAINT PK_Roles PRIMARY KEY,
    RoleName varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS dbo.Users (
    Id uuid NOT NULL CONSTRAINT PK_Users PRIMARY KEY,
    Fullname varchar(255) NOT NULL,
    UserName varchar(255) NOT NULL,
    Email varchar(255) NOT NULL,
    Password text NOT NULL
);

CREATE TABLE IF NOT EXISTS dbo.UserRole (
    UserId uuid NOT NULL CONSTRAINT PK_UserRole PRIMARY KEY,
    RoleId uuid NOT NULL,
    CONSTRAINT FK_UserRole_Users_UserId FOREIGN KEY (UserId) REFERENCES dbo.Users (Id) ON DELETE CASCADE,
    CONSTRAINT FK_UserRole_Roles_RoleId FOREIGN KEY (RoleId) REFERENCES dbo.Roles (Id)
);

CREATE TABLE IF NOT EXISTS dbo.UserAccess (
    Id uuid NOT NULL CONSTRAINT PK_UserAccess PRIMARY KEY,
    UserId uuid NOT NULL,
    Modul varchar(255) NOT NULL,
    is_add boolean NOT NULL DEFAULT false,
    is_edit boolean NOT NULL DEFAULT false,
    is_delete boolean NOT NULL DEFAULT false,
    is_view boolean NOT NULL DEFAULT false,
    is_download boolean NOT NULL DEFAULT false,
    CONSTRAINT FK_UserAccess_Users_UserId FOREIGN KEY (UserId) REFERENCES dbo.Users (Id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.MasterManufacturer (
    vendor_id varchar(50) NOT NULL CONSTRAINT PK_MasterManufacturer PRIMARY KEY,
    manufacturer_name varchar(255) NOT NULL,
    logo_image text NULL,
    valid_until timestamp NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL
);

CREATE TABLE IF NOT EXISTS dbo.MasterComponent (
    component_id varchar(50) NOT NULL CONSTRAINT PK_MasterComponent PRIMARY KEY,
    component_name varchar(255) NOT NULL,
    vendor_id varchar(50) NOT NULL,
    failure_rate numeric(38, 18) NULL,
    cost text NULL,
    compatibility text NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    serial_number varchar(255) NULL,
    CONSTRAINT FK_MasterComponent_MasterManufacturer_vendor_id FOREIGN KEY (vendor_id) REFERENCES dbo.MasterManufacturer (vendor_id)
);

CREATE TABLE IF NOT EXISTS dbo.MasterProject (
    project_id varchar(50) NOT NULL CONSTRAINT PK_MasterProject PRIMARY KEY,
    project_name varchar(255) NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL
);

CREATE TABLE IF NOT EXISTS dbo.RbdSystemDrawing (
    rbd_system_id varchar(50) NOT NULL CONSTRAINT PK_RbdSystemDrawing PRIMARY KEY,
    project_id varchar(50) NOT NULL,
    drawing_name varchar(255) NULL,
    system_name varchar(255) NULL,
    running_hours varchar(255) NULL,
    reliability_total numeric(38, 18) NULL,
    formula text NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    CONSTRAINT FK_RbdSystemDrawing_MasterProject_project_id FOREIGN KEY (project_id) REFERENCES dbo.MasterProject (project_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.SystemComponentProperties (
    system_component_id varchar(50) NOT NULL CONSTRAINT PK_SystemComponentProperties PRIMARY KEY,
    rbd_system_id varchar(50) NULL,
    parent_id varchar(20) NOT NULL DEFAULT '',
    component_name varchar(255) NOT NULL,
    component_tag_number varchar(255) NULL,
    active int NULL,
    vendor varchar(255) NULL,
    formula_code varchar(20) NULL,
    distribution_type varchar(50) NULL,
    failure_rate numeric(38, 18) NULL,
    running_hours numeric(38, 18) NULL,
    scale_parameter numeric(38, 18) NULL,
    shape_parameter numeric(38, 18) NULL,
    connection_type varchar(50) NULL,
    connection_to_id varchar(255) NULL,
    position_x varchar(255) NULL,
    position_y varchar(255) NULL,
    source_position varchar(50) NULL,
    target_position varchar(50) NULL,
    id_node varchar(255) NULL,
    reliability_value numeric(38, 18) NULL,
    active_component int NULL,
    total_component int NULL,
    regresi numeric(38, 18) NULL,
    mtbf numeric(38, 18) NULL,
    created_at timestamp NULL DEFAULT now(),
    updated_at timestamp NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    min_unit int NULL DEFAULT 1,
    max_unit int NULL DEFAULT 5
);

CREATE UNIQUE INDEX IF NOT EXISTS IX_SystemComponentProperties_connection_to_id ON dbo.SystemComponentProperties (connection_to_id) WHERE connection_to_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS IX_SystemComponentProperties_parent_id ON dbo.SystemComponentProperties (parent_id);

CREATE INDEX IF NOT EXISTS IX_SystemComponentProperties_rbd_system_id ON dbo.SystemComponentProperties (rbd_system_id);

CREATE TABLE IF NOT EXISTS dbo.SystemComponentDrawing (
    id_edge varchar(255) NOT NULL CONSTRAINT PK_SystemComponentDrawing PRIMARY KEY,
    source_id varchar(255) NULL,
    target_id varchar(255) NULL
);

CREATE TABLE IF NOT EXISTS dbo.FailureEventHistory (
    failure_event_id varchar(50) NOT NULL CONSTRAINT PK_FailureEventHistory PRIMARY KEY,
    system_component_id varchar(50) NOT NULL,
    failure_date timestamp NOT NULL,
    failure_number int NULL,
    running_hours int NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    CONSTRAINT FK_FailureEventHistory_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.WeibullParameter (
    weibull_parameter_id varchar(50) NOT NULL CONSTRAINT PK_WeibullParameter PRIMARY KEY,
    system_component_id varchar(50) NOT NULL,
    failure_event_hours int NOT NULL,
    n int NOT NULL,
    freq_f numeric(38, 18) NOT NULL,
    x numeric(38, 18) NOT NULL,
    y numeric(38, 18) NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    CONSTRAINT FK_WeibullParameter_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.ExponentialParameter (
    exponential_parameter_id varchar(50) NOT NULL CONSTRAINT PK_ExponentialParameter PRIMARY KEY,
    system_component_id varchar(50) NOT NULL,
    failure_event_hours int NOT NULL,
    n int NOT NULL,
    fregf numeric(38, 18) NOT NULL,
    f_t_median_rank numeric(38, 18) NOT NULL,
    r_t numeric(38, 18) NOT NULL,
    inRt int NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL
);

CREATE TABLE IF NOT EXISTS dbo.ReliabilityPlotComponent (
    reliability_plot_id varchar(50) NOT NULL CONSTRAINT PK_ReliabilityPlotComponent PRIMARY KEY,
    system_component_id varchar(50) NOT NULL,
    time_t int NOT NULL,
    reliability_comp numeric(38, 18) NULL,
    created_at timestamp NULL DEFAULT now(),
    updated_at timestamp NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    CONSTRAINT FK_ReliabilityPlotComponent_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.Hierarchy (
    hierarchy_id varchar(20) NOT NULL CONSTRAINT PK_Hierarchy PRIMARY KEY,
    rbd_system_id varchar(50) NULL,
    parent_id varchar(20) NOT NULL,
    level int NOT NULL,
    sub_system_name varchar(255) NULL,
    formula text NULL,
    formula_code varchar(20) NULL,
    connection_type varchar(255) NULL,
    realibility_value numeric(38, 18) NULL,
    running_hours int NULL,
    position_x numeric(38, 18) NULL,
    position_y numeric(38, 18) NULL,
    source_id varchar(255) NULL,
    target_id varchar(255) NULL,
    CONSTRAINT FK_Hierarchy_RbdSystemDrawing_rbd_system_id FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS IX_Hierarchy_rbd_system_id ON dbo.Hierarchy (rbd_system_id);

CREATE INDEX IF NOT EXISTS IX_Hierarchy_parent_id ON dbo.Hierarchy (parent_id);

CREATE INDEX IF NOT EXISTS IX_Hierarchy_level ON dbo.Hierarchy (level);

CREATE TABLE IF NOT EXISTS dbo.ReliabilityHistory (
    history_id varchar(50) NOT NULL CONSTRAINT PK_ReliabilityHistory PRIMARY KEY,
    rbd_system_id varchar(50) NOT NULL,
    hierarchy_id varchar(20) NOT NULL,
    hierarchy_name varchar(255) NULL,
    hierarchy_level int NULL,
    formula_code varchar(20) NULL,
    formula text NULL,
    calculated_reliability numeric(38, 18) NULL,
    reliability_lookup text NULL,
    component_details text NULL,
    running_hours numeric(38, 18) NULL,
    calculation_timestamp timestamp NOT NULL DEFAULT now(),
    calculated_by varchar(100) NULL,
    CONSTRAINT FK_ReliabilityHistory_RbdSystem FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id) ON DELETE CASCADE,
    CONSTRAINT FK_ReliabilityHistory_Hierarchy FOREIGN KEY (hierarchy_id) REFERENCES dbo.Hierarchy (hierarchy_id)
);

CREATE INDEX IF NOT EXISTS IX_ReliabilityHistory_HierarchyId ON dbo.ReliabilityHistory (hierarchy_id, calculation_timestamp DESC);

CREATE INDEX IF NOT EXISTS IX_ReliabilityHistory_RbdSystemHierarchy ON dbo.ReliabilityHistory (rbd_system_id, hierarchy_id, calculation_timestamp DESC);

CREATE INDEX IF NOT EXISTS IX_ReliabilityHistory_Timestamp ON dbo.ReliabilityHistory (calculation_timestamp DESC);

CREATE TABLE IF NOT EXISTS dbo.OptimizationRun (
    optimization_run_id varchar(50) NOT NULL CONSTRAINT PK_OptimizationRun PRIMARY KEY,
    rbd_system_id varchar(50) NOT NULL,
    optimization_type int NOT NULL,
    target_reliability numeric(18, 10) NULL,
    max_budget numeric(18, 2) NULL,
    weight_cost numeric(5, 4) NULL,
    weight_reliability numeric(5, 4) NULL,
    running_hours numeric(18, 4) NOT NULL DEFAULT 1000,
    population_size int NOT NULL DEFAULT 50,
    max_generations int NOT NULL DEFAULT 100,
    crossover_probability numeric(5, 4) NOT NULL DEFAULT 0.8,
    mutation_probability numeric(5, 4) NOT NULL DEFAULT 0.1,
    status varchar(20) NOT NULL DEFAULT 'Running',
    best_fitness numeric(18, 10) NULL,
    best_reliability numeric(18, 10) NULL,
    best_cost numeric(18, 2) NULL,
    generations_completed int NULL,
    execution_time_ms bigint NULL,
    error_message text NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    completed_at timestamp NULL,
    created_by varchar(100) NULL,
    CONSTRAINT FK_OptimizationRun_RbdSystem FOREIGN KEY (rbd_system_id) REFERENCES dbo.RbdSystemDrawing (rbd_system_id)
);

CREATE TABLE IF NOT EXISTS dbo.OptimizationResult (
    optimization_result_id varchar(50) NOT NULL CONSTRAINT PK_OptimizationResult PRIMARY KEY,
    optimization_run_id varchar(50) NOT NULL,
    system_component_id varchar(50) NOT NULL,
    component_name varchar(200) NOT NULL,
    original_vendor_id varchar(50) NULL,
    optimized_vendor_id varchar(50) NOT NULL,
    optimized_vendor_name varchar(200) NULL,
    original_unit_count int NULL,
    optimized_unit_count int NOT NULL DEFAULT 1,
    connection_type varchar(50) NULL,
    failure_rate numeric(18, 10) NULL,
    cost_per_unit numeric(18, 2) NULL,
    total_cost numeric(18, 2) NULL,
    component_reliability numeric(18, 10) NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT FK_OptimizationResult_Run FOREIGN KEY (optimization_run_id) REFERENCES dbo.OptimizationRun (optimization_run_id)
);

CREATE INDEX IF NOT EXISTS IX_OptimizationRun_RbdSystemId ON dbo.OptimizationRun (rbd_system_id);

CREATE INDEX IF NOT EXISTS IX_OptimizationResult_RunId ON dbo.OptimizationResult (optimization_run_id);
