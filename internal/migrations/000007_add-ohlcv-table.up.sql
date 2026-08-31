CREATE TABLE
    ohlcv (
        "time" TIMESTAMPTZ NOT NULL,
        "symbol" TEXT NOT NULL,
        "open" NUMERIC(18, 8) NOT NULL,
        "high" NUMERIC(18, 8) NOT NULL,
        "low" NUMERIC(18, 8) NOT NULL,
        "close" NUMERIC(18, 8) NOT NULL,
        "volume" NUMERIC(18, 8) NOT NULL
    );

SELECT
    create_hypertable ('ohlcv', 'time', migrate_data => true);

CREATE INDEX "ohlcv_symbol_time_idx" ON ohlcv ("symbol", "time" DESC);