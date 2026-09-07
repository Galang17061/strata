CREATE TABLE IF NOT EXISTS dbo.PoissonParameter (
    poisson_parameter_id varchar(50) NOT NULL CONSTRAINT PK_PoissonParameter PRIMARY KEY,
    system_component_id varchar(50) NOT NULL,
    failure_event_hours int NOT NULL,
    n int NOT NULL,
    rate numeric(38, 18) NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NOT NULL DEFAULT now(),
    created_by varchar(100) NULL,
    updated_by varchar(100) NULL,
    CONSTRAINT FK_PoissonParameter_SystemComponentProperties_system_component_id FOREIGN KEY (system_component_id) REFERENCES dbo.SystemComponentProperties (system_component_id) ON DELETE CASCADE
);
