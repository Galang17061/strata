CREATE TABLE IF NOT EXISTS dbo.PasswordReset (
    password_reset_id varchar(50) NOT NULL CONSTRAINT PK_PasswordReset PRIMARY KEY,
    user_id uuid NOT NULL,
    token_hash varchar(100) NOT NULL,
    expires_at timestamp NOT NULL,
    used_at timestamp NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT FK_PasswordReset_Users_user_id FOREIGN KEY (user_id) REFERENCES dbo.Users (Id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dbo.UserInvite (
    user_invite_id varchar(50) NOT NULL CONSTRAINT PK_UserInvite PRIMARY KEY,
    email varchar(256) NOT NULL,
    role_id uuid NOT NULL,
    token_hash varchar(100) NOT NULL,
    expires_at timestamp NOT NULL,
    accepted_at timestamp NULL,
    created_by varchar(100) NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT FK_UserInvite_Roles_role_id FOREIGN KEY (role_id) REFERENCES dbo.Roles (Id)
);
