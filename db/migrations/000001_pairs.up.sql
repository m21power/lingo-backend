CREATE TABLE pairs (
    id VARCHAR(50) PRIMARY KEY, -- Custom string ID
    user1id BIGINT NOT NULL,
    user2id BIGINT NOT NULL,
    user3id BIGINT, -- optional third user
    username1 VARCHAR(255) NOT NULL,
    username2 VARCHAR(255) NOT NULL,
    username3 VARCHAR(255), -- optional third user
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Prevent same group from being duplicated on same day
    CONSTRAINT unique_pair_per_day UNIQUE (user1id, user2id, user3id, date)
);
