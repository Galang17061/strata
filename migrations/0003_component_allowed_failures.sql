ALTER TABLE dbo.SystemComponentProperties ADD COLUMN IF NOT EXISTS allowed_failures int NOT NULL DEFAULT 0;
