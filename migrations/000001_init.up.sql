CREATE TABLE exchange_rates (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    currency    CHAR(3)        NOT NULL,
    rate        DECIMAL(18, 6) NOT NULL,
    source_date DATETIME(3)    NOT NULL,
    fetched_at  DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uniq_currency_date (currency, source_date),
    INDEX idx_currency_fetched (currency, fetched_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
