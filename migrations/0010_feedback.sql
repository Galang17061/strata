IF OBJECT_ID(N'dbo.Feedback', N'U') IS NULL
CREATE TABLE dbo.Feedback (
    feedback_id NVARCHAR(50) NOT NULL CONSTRAINT PK_Feedback PRIMARY KEY,
    user_name NVARCHAR(255) NOT NULL,
    category NVARCHAR(20) NOT NULL,
    message NVARCHAR(2000) NOT NULL,
    page NVARCHAR(400) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_Feedback_created_at DEFAULT GETDATE()
);
GO
