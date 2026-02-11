CREATE TABLE notifications (
    id VARCHAR(255) PRIMARY KEY,
    message TEXT NOT NULL,
    time VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    chat_id BIGINT NOT NULL
);
