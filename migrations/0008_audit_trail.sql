CREATE TABLE IF NOT EXISTS dbo.AuditTrail (
    audit_trail_id varchar(50) NOT NULL CONSTRAINT PK_AuditTrail PRIMARY KEY,
    user_name varchar(255) NOT NULL,
    method varchar(10) NOT NULL,
    path varchar(400) NOT NULL,
    status_code int NOT NULL,
    created_at timestamp NOT NULL DEFAULT now()
);
