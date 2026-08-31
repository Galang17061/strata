IF COL_LENGTH(N'dbo.SystemComponentProperties', N'allowed_failures') IS NULL
ALTER TABLE dbo.SystemComponentProperties ADD allowed_failures INT NOT NULL CONSTRAINT DF_SystemComponentProperties_allowed_failures DEFAULT 0 WITH VALUES;
GO
