CREATE OR REPLACE VIEW {{ .InstrumentType }}_{{ .Quantity }}_{{ .TimeUnit }}_volume_changes AS
SELECT 
    "time", 
    symbol, 
    ((close - previous_volume_change) / previous_volume_change) * 100 AS volume_change
FROM (
    SELECT 
        *, 
        LAG(volume) OVER(PARTITION BY symbol ORDER BY "time") AS previous_volume 
    FROM {{ .ReferenceTable }}
) 
WHERE previous_volume_change IS NOT NULL;
