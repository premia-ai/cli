CREATE MATERIALIZED VIEW stocks_{{ .Quantity }}_{{ .TimeUnit }}_candles
WITH (timescaledb.continuous) AS
    SELECT
        time_bucket('{{ .Quantity }} {{ .TimeUnit }}', time) AS bucket,
        symbol,
        FIRST(open, time) AS "open",
        MAX(high) AS high,
        MIN(low) AS low,
        LAST(close, time) AS "close",
        SUM(volume) AS volume,
        currency
    FROM {{ .ReferenceTable }}
    GROUP BY bucket, currency, symbol
WITH NO DATA;

SELECT add_continuous_aggregate_policy('stocks_{{ .Quantity }}_{{ .TimeUnit }}_candles',
    start_offset => INTERVAL '3 day',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');
