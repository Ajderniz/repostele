-- Seeds a handful of placeholder menu items for local testing.
-- Usage: sqlite3 bin/data.db < scripts/seed_items.sql
-- Safe to re-run: items whose name already exists are skipped.

INSERT INTO "items" ("name", "price", "time_mod", "available", "desc", "img_path")
SELECT s."name", s."price", strftime('%s','now'), s."available", s."desc", ''
FROM (
  SELECT 'Pan dulce' AS "name", 850 AS "price", 1 AS "available", 'Pan dulce tradicional recién horneado' AS "desc"
  UNION ALL SELECT 'Queque de naranja',  2200, 1, 'Queque húmedo con ralladura de naranja'
  UNION ALL SELECT 'Empanada de queso',  950,  1, 'Empanada rellena de queso, horneada'
  UNION ALL SELECT 'Tres leches',        1800, 1, 'Porción individual de tres leches'
  UNION ALL SELECT 'Galletas de avena',  600,  1, 'Seis unidades por orden'
  UNION ALL SELECT 'Torta de chocolate', 3500, 0, 'Porción de torta de chocolate (agotada)'
) AS s
WHERE NOT EXISTS (SELECT 1 FROM "items" i WHERE i."name" = s."name");
