CREATE TABLE pairs (
    id SERIAL PRIMARY KEY,
    user1id BIGINT NOT NULL,
    user2id BIGINT NOT NULL,
    userid3 BIGINT, -- nullable, optional third user
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Enforce that the same group doesn't repeat on the same date
    CONSTRAINT unique_pair_per_day UNIQUE (user1id, user2id, userid3, date)
);
