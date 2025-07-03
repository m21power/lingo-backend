CREATE TABLE notification_seen (
    notificationId VARCHAR(255) NOT NULL,
    userId BIGINT NOT NULL,
    seenAt TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (notificationId, userId),
    FOREIGN KEY (notificationId) REFERENCES notifications(id)
);