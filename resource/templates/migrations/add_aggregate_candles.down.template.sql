-- this also drops the related continuous aggregrate policy
DROP MATERIALIZED VIEW IF EXISTS stocks_{{ .Quantity }}_{{ .TimeUnit }}_candles;
