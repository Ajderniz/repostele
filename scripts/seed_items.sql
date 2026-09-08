-- Seeds a handful of placeholder menu items for local testing.
-- Usage: sqlite3 bin/data.db < scripts/seed_items.sql

INSERT INTO "items" ("name", "price", "time_mod", "available", "desc", "img_path")
VALUES
  ('Pan dulce',        850,  strftime('%s','now'), 1, 'Pan dulce tradicional recién horneado', ''),
  ('Queque de naranja', 2200, strftime('%s','now'), 1, 'Queque húmedo con ralladura de naranja', ''),
  ('Empanada de queso', 950,  strftime('%s','now'), 1, 'Empanada rellena de queso, horneada', ''),
  ('Tres leches',      1800, strftime('%s','now'), 1, 'Porción individual de tres leches', ''),
  ('Galletas de avena', 600,  strftime('%s','now'), 1, 'Seis unidades por orden', ''),
  ('Torta de chocolate', 3500, strftime('%s','now'), 0, 'Porción de torta de chocolate (agotada)', '');
