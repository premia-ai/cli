DROP INDEX IF EXISTS stocks_{{ .Quantity }}_{{ .TimeUnit }}_candles_symbol_time_idx;
DROP TABLE IF EXISTS stocks_{{ .Quantity }}_{{ .TimeUnit }}_candles;
