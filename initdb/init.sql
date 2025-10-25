CREATE TABLE IF NOT EXISTS orders_json (
  order_uid TEXT PRIMARY KEY,
  data JSONB NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT now()
);