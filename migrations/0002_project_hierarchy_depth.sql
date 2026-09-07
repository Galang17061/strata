ALTER TABLE dbo.MasterProject ADD COLUMN IF NOT EXISTS hierarchy_depth int NOT NULL DEFAULT 3;
