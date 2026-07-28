CREATE TABLE IF NOT EXISTS "items" (
	"id"	INTEGER,
	"name"	TEXT NOT NULL,
	"price"	NUMERIC NOT NULL DEFAULT 0 CHECK(0 <= "price"),
	"time_mod"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "time_mod"),
	"available"	INTEGER NOT NULL DEFAULT 1 CHECK("available" IN (0, 1)),
	"desc"	TEXT,
	"img_path"	TEXT,
	PRIMARY KEY("id")
);
CREATE TABLE IF NOT EXISTS "order_items" (
	"id"	INTEGER,
	"order_id"	INTEGER NOT NULL,
	"item_id"	INTEGER NOT NULL,
	"quant"	INTEGER NOT NULL DEFAULT 1 CHECK(1 <= "quant" AND "quant" <= 4),
	PRIMARY KEY("id"),
	FOREIGN KEY("item_id") REFERENCES "items"("id"),
	FOREIGN KEY("order_id") REFERENCES "orders"("id")
);
CREATE TABLE IF NOT EXISTS "orders" (
	"id"	INTEGER,
	"user"	TEXT NOT NULL DEFAULT '',
	"total"	NUMERIC NOT NULL CHECK(0 <= "total"),
	"ref_num"	TEXT NOT NULL DEFAULT '' CHECK(LENGTH("ref_num") == 25) UNIQUE,
	"time"	INTEGER NOT NULL CHECK(0 <= "time"),
	"status"	INTEGER NOT NULL CHECK(0 <= "status" AND "status" <= 4),
	"updated"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "updated"),
	PRIMARY KEY("id"),
	FOREIGN KEY("user") REFERENCES "users"("username")
);
CREATE TABLE IF NOT EXISTS "sessions" (
	"session_token"	TEXT,
	"csrf_token"	TEXT NOT NULL DEFAULT '',
	"user"	TEXT NOT NULL DEFAULT '',
	"role"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "role" AND "role" <= 1),
	"starts"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "starts"),
	"expires"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "expires"),
	PRIMARY KEY("session_token"),
	FOREIGN KEY("user") REFERENCES "users"("username")
);
CREATE TABLE IF NOT EXISTS "staff" (
	"username"	TEXT CHECK(4 <= LENGTH("username") AND LENGTH("username") <= 16),
	"pass_hash"	TEXT NOT NULL UNIQUE,
	"full_name"	TEXT NOT NULL,
	"time_created"	INTEGER NOT NULL DEFAULT 0 CHECK(0 <= "time_created"),
	"active"	INTEGER NOT NULL DEFAULT 1 CHECK("active" IN (0, 1)),
	"admin"	INTEGER NOT NULL DEFAULT 0 CHECK("admin" IN (0, 1)),
	PRIMARY KEY("username")
);
CREATE TABLE IF NOT EXISTS "users" (
	"username"	TEXT CHECK(4 <= LENGTH("username") AND LENGTH("username") <= 16),
	"pass_hash"	TEXT NOT NULL UNIQUE,
	"time_created"	INTEGER NOT NULL,
	"active"	INTEGER NOT NULL DEFAULT 1 CHECK("active" IN (0, 1)),
	PRIMARY KEY("username")
);
