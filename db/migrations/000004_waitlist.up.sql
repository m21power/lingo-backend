CREATE TABLE waitlist(
    id SERIAL PRIMARY KEY,
    userId BIGINT NOT NULL,
    username VARCHAR(255) NOT NULL,
    profileUrl VARCHAR(255),
    createdAt TIMESTAMP NOT NULL DEFAULT NOW()
);