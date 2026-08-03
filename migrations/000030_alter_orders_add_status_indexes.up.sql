CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_status_histories_to_status_created_at ON order_status_histories(to_status, created_at);
