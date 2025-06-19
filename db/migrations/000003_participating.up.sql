CREATE TABLE pair_participation (
    id SERIAL PRIMARY KEY,
    pair_id INT REFERENCES pairs(id) ON DELETE CASCADE,
    userid BIGINT NOT NULL,
    is_participating BOOLEAN DEFAULT NULL, -- NULL = not responded yet
    responded_at TIMESTAMP
);
