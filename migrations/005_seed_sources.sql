INSERT INTO sources (name, base_url, mode, active, scrape_interval_minutes)
SELECT 'dealls', 'https://dealls.com/', 'http', TRUE, 60
WHERE NOT EXISTS (
    SELECT 1 FROM sources WHERE name = 'dealls'
);

INSERT INTO sources (name, base_url, mode, active, scrape_interval_minutes)
SELECT 'glints', 'https://glints.com/', 'browser', TRUE, 60
WHERE NOT EXISTS (
    SELECT 1 FROM sources WHERE name = 'glints'
);
