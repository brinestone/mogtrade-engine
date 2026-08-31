CREATE MATERIALIZED VIEW
    ohlcv_1h
WITH
    (timescaledb.continuous) AS
SELECT
    time_bucket ('1 hour', "time") AS bucket,
    symbol,
    first (open, time) AS open,
    max(high) as high,
    min(low) as low,
    last ("close", time) as "close",
    sum("volume") as volume
FROM
    ohlcv
GROUP BY
    bucket,
    symbol;