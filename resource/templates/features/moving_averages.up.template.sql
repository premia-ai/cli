CREATE OR REPLACE VIEW stocks_{{ .Quantity }}_{{ .TimeUnit }}_averages AS
SELECT time, symbol, average
FROM (
     SELECT
        time,
        symbol,
        AVG(close) OVER (
            PARTITION BY symbol 
            ORDER BY time 
            ROWS BETWEEN {{ sub .Quantity 1 }} PRECEDING AND CURRENT ROW
        ) AS average,
        COUNT(close) OVER (
            PARTITION BY symbol 
            ORDER BY time 
            ROWS BETWEEN {{ sub .Quantity 1 }} PRECEDING AND CURRENT ROW
        ) AS row_count
    FROM {{ .ReferenceTable }}
)
WHERE row_count = {{ .Quantity }};
