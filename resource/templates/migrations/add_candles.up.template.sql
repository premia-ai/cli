CREATE TABLE IF NOT EXISTS {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles (
    time TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    open NUMERIC NULL,
    close NUMERIC NULL,
    high NUMERIC NULL,
    low NUMERIC NULL,
    volume INT NULL,
    currency TEXT NOT NULL,
    data_provider TEXT NOT NULL
);

SELECT create_hypertable('{{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles', by_range('time'));

CREATE UNIQUE INDEX IF NOT EXISTS {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles_symbol_time_idx
ON {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles (symbol, time DESC);
