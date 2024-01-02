-- this also drops the related continuous aggregrate policy
DROP MATERIALIZED VIEW IF EXISTS {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_candles;
