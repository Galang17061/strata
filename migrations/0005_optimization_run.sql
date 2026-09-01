IF OBJECT_ID(N'dbo.OptimizationRun', N'U') IS NULL
CREATE TABLE dbo.OptimizationRun (
    optimization_run_id NVARCHAR(50) NOT NULL CONSTRAINT PK_OptimizationRun PRIMARY KEY,
    rbd_system_id NVARCHAR(50) NOT NULL,
    mode INT NOT NULL,
    max_budget FLOAT NULL,
    target_reliability FLOAT NULL,
    weight_cost FLOAT NULL,
    weight_reliability FLOAT NULL,
    running_hours FLOAT NULL,
    population_size INT NULL,
    max_generations INT NULL,
    crossover_probability FLOAT NULL,
    mutation_probability FLOAT NULL,
    seed BIGINT NULL,
    choices NVARCHAR(MAX) NULL,
    result_project_id NVARCHAR(50) NULL,
    result_rbd_system_id NVARCHAR(50) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_OptimizationRun_created_at DEFAULT GETDATE(),
    created_by NVARCHAR(100) NULL
);
GO
