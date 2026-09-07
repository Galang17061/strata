DO $$
DECLARE
    has_rows boolean;
BEGIN
    IF to_regclass('dbo.OptimizationResult') IS NOT NULL THEN
        EXECUTE 'SELECT EXISTS (SELECT 1 FROM dbo.OptimizationResult)' INTO has_rows;
        IF NOT has_rows THEN
            DROP TABLE dbo.OptimizationResult;
        END IF;
    END IF;
    IF to_regclass('dbo.OptimizationRun') IS NOT NULL AND to_regclass('dbo.OptimizationResult') IS NULL THEN
        EXECUTE 'SELECT EXISTS (SELECT 1 FROM dbo.OptimizationRun)' INTO has_rows;
        IF NOT has_rows THEN
            DROP TABLE dbo.OptimizationRun;
        END IF;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS dbo.OptimizationRun (
    optimization_run_id varchar(50) NOT NULL CONSTRAINT PK_OptimizationRun PRIMARY KEY,
    rbd_system_id varchar(50) NOT NULL,
    mode int NOT NULL,
    max_budget double precision NULL,
    target_reliability double precision NULL,
    weight_cost double precision NULL,
    weight_reliability double precision NULL,
    running_hours double precision NULL,
    population_size int NULL,
    max_generations int NULL,
    crossover_probability double precision NULL,
    mutation_probability double precision NULL,
    seed bigint NULL,
    choices text NULL,
    result_project_id varchar(50) NULL,
    result_rbd_system_id varchar(50) NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL
);

ALTER TABLE dbo.OptimizationRun ADD COLUMN IF NOT EXISTS mode int NOT NULL DEFAULT 0;
ALTER TABLE dbo.OptimizationRun ADD COLUMN IF NOT EXISTS seed bigint NULL;
ALTER TABLE dbo.OptimizationRun ADD COLUMN IF NOT EXISTS choices text NULL;
ALTER TABLE dbo.OptimizationRun ADD COLUMN IF NOT EXISTS result_project_id varchar(50) NULL;
ALTER TABLE dbo.OptimizationRun ADD COLUMN IF NOT EXISTS result_rbd_system_id varchar(50) NULL;
