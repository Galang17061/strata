IF OBJECT_ID(N'dbo.OptimizationResult', N'U') IS NOT NULL AND NOT EXISTS (SELECT 1 FROM dbo.OptimizationResult)
DROP TABLE dbo.OptimizationResult;
GO

IF OBJECT_ID(N'dbo.OptimizationRun', N'U') IS NOT NULL AND OBJECT_ID(N'dbo.OptimizationResult', N'U') IS NULL AND NOT EXISTS (SELECT 1 FROM dbo.OptimizationRun)
DROP TABLE dbo.OptimizationRun;
GO

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

IF COL_LENGTH(N'dbo.OptimizationRun', N'mode') IS NULL
ALTER TABLE dbo.OptimizationRun ADD mode INT NOT NULL CONSTRAINT DF_OptimizationRun_mode DEFAULT 0 WITH VALUES;
GO
IF COL_LENGTH(N'dbo.OptimizationRun', N'seed') IS NULL
ALTER TABLE dbo.OptimizationRun ADD seed BIGINT NULL;
GO
IF COL_LENGTH(N'dbo.OptimizationRun', N'choices') IS NULL
ALTER TABLE dbo.OptimizationRun ADD choices NVARCHAR(MAX) NULL;
GO
IF COL_LENGTH(N'dbo.OptimizationRun', N'result_project_id') IS NULL
ALTER TABLE dbo.OptimizationRun ADD result_project_id NVARCHAR(50) NULL;
GO
IF COL_LENGTH(N'dbo.OptimizationRun', N'result_rbd_system_id') IS NULL
ALTER TABLE dbo.OptimizationRun ADD result_rbd_system_id NVARCHAR(50) NULL;
GO
