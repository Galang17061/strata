IF COL_LENGTH(N'dbo.MasterProject', N'hierarchy_depth') IS NULL
ALTER TABLE dbo.MasterProject ADD hierarchy_depth INT NOT NULL CONSTRAINT DF_MasterProject_hierarchy_depth DEFAULT 3 WITH VALUES;
GO
