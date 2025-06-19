CREATE TABLE pairs (
    id SERIAL PRIMARY KEY,
    user1id BIGINT NOT NULL,
    user2id BIGINT NOT NULL,
    user3id BIGINT, -- nullable, optional third user
    username1 VARCHAR(255) NOT NULL,
    username2 VARCHAR(255) NOT NULL,
    username3 VARCHAR(255), -- nullable, optional third user
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Enforce that the same group doesn't repeat on the same date
    CONSTRAINT unique_pair_per_day UNIQUE (user1id, user2id, user3id, date)
);
