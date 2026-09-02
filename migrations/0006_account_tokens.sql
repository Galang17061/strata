IF OBJECT_ID(N'dbo.PasswordReset', N'U') IS NULL
CREATE TABLE dbo.PasswordReset (
    password_reset_id NVARCHAR(50) NOT NULL CONSTRAINT PK_PasswordReset PRIMARY KEY,
    user_id UNIQUEIDENTIFIER NOT NULL,
    token_hash NVARCHAR(100) NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_PasswordReset_created_at DEFAULT GETDATE(),
    CONSTRAINT FK_PasswordReset_Users_user_id FOREIGN KEY (user_id) REFERENCES dbo.Users (Id) ON DELETE CASCADE
);
GO
IF OBJECT_ID(N'dbo.UserInvite', N'U') IS NULL
CREATE TABLE dbo.UserInvite (
    user_invite_id NVARCHAR(50) NOT NULL CONSTRAINT PK_UserInvite PRIMARY KEY,
    email NVARCHAR(256) NOT NULL,
    role_id UNIQUEIDENTIFIER NOT NULL,
    token_hash NVARCHAR(100) NOT NULL,
    expires_at DATETIME NOT NULL,
    accepted_at DATETIME NULL,
    created_by NVARCHAR(100) NULL,
    created_at DATETIME NOT NULL CONSTRAINT DF_UserInvite_created_at DEFAULT GETDATE(),
    CONSTRAINT FK_UserInvite_Roles_role_id FOREIGN KEY (role_id) REFERENCES dbo.Roles (Id)
);
GO
