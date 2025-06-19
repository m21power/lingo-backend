CREATE TABLE otp (
    id SERIAL PRIMARY KEY,
    userid BIGINT NOT NULL,
    username VARCHAR(100) NOT NULL,
    otp BIGINT NOT NULL,
    createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_user_otp UNIQUE (userid, username)
);
