-- schema.sql
CREATE TABLE IF NOT EXISTS deliveries (
  id SERIAL PRIMARY KEY,
  name TEXT,
  phone TEXT,
  zip TEXT,
  city TEXT,
  address TEXT,
  region TEXT,
  email TEXT
);

CREATE TABLE IF NOT EXISTS payments (
  id SERIAL PRIMARY KEY,
  transaction TEXT,
  request_id TEXT,
  currency TEXT,
  provider TEXT,
  amount BIGINT,
  payment_dt BIGINT,
  bank TEXT,
  delivery_cost BIGINT,
  goods_total BIGINT,
  custom_fee BIGINT
);

CREATE TABLE IF NOT EXISTS orders (
  order_uid TEXT PRIMARY KEY,
  track_number TEXT,
  entry TEXT,
  delivery_id INTEGER REFERENCES deliveries(id) ON DELETE SET NULL,
  payment_id INTEGER REFERENCES payments(id) ON DELETE SET NULL,
  locale TEXT,
  internal_signature TEXT,
  customer_id TEXT,
  delivery_service TEXT,
  shardkey TEXT,
  sm_id INTEGER,
  date_created TIMESTAMP WITH TIME ZONE,
  oof_shard TEXT
);

CREATE TABLE IF NOT EXISTS items (
  id SERIAL PRIMARY KEY,
  order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE,
  chrt_id BIGINT,
  track_number TEXT,
  price BIGINT,
  rid TEXT,
  name TEXT,
  sale INTEGER,
  size TEXT,
  total_price BIGINT,
  nm_id BIGINT,
  brand TEXT,
  status INTEGER
);

CREATE TABLE IF NOT EXISTS orders_json (
  order_uid TEXT PRIMARY KEY,
  data JSONB NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_json_order_uid ON orders_json(order_uid);

-- индекс для быстрого поиска by order_uid (в items — уже есть order_uid FK)
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
