CREATE TABLE IF NOT EXISTS dbo.Feedback (
    feedback_id varchar(50) NOT NULL CONSTRAINT PK_Feedback PRIMARY KEY,
    user_name varchar(255) NOT NULL,
    category varchar(20) NOT NULL,
    message varchar(2000) NOT NULL,
    page varchar(400) NULL,
    created_at timestamp NOT NULL DEFAULT now()
);
