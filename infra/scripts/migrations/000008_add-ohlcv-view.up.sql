CREATE MATERIALIZED VIEW ohlcv_1h AS
SELECT
    time_bucket('1 hour', "time") AS bucket,
    symbol,
    first(open, time) AS open,
    max(high) AS high,
    min(low) AS low,
    last("close", time) AS "close",
    sum("volume") AS volume
FROM ohlcv
GROUP BY
    bucket,
    symbol;