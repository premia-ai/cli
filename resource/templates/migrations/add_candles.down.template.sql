DROP INDEX IF EXISTS {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles_symbol_time_idx;
DROP TABLE IF EXISTS {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles;
