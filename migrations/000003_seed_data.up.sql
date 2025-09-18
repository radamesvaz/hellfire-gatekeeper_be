DELETE FROM products_history WHERE id_product IN (1, 2);
DELETE FROM products WHERE id_product IN (1, 2);
DELETE FROM users WHERE id_user IN (1, 2);
DELETE FROM roles WHERE id_role IN (1, 2);

INSERT INTO roles (id_role, name) VALUES
  (1, 'admin'),
  (2, 'client');

INSERT INTO users (id_role, name, email, phone, password_hash, created_on)
SELECT 
  (SELECT id_role FROM roles WHERE name = 'admin'),
  'Admin User',
  'admin@example.com',
  '55-55555',
  '$2a$10$SO6GoHITlrEuH9mZyaFqN.vmtx9F0mn.4c2Blp7xM9WCM1svOMnYq',
  '2025-04-14 10:00:00'::timestamp
UNION ALL
SELECT 
  (SELECT id_role FROM roles WHERE name = 'client'),
  'Client',
  'client@example.com',
  '66-6666',
  NULL,
  '2025-04-14 10:00:00'::timestamp;

INSERT INTO products (name, description, price, available, created_on)
VALUES
  ('Brownie Clásico', 'Delicioso brownie de chocolate', 3.5, true, '2025-04-14 10:00:00'::timestamp),
  ('Suspiros', 'Suspiros tradicionales', 5, true, '2025-04-14 10:00:00'::timestamp);

-- Insert product history after products are created
INSERT INTO products_history (id_product, name, description, price, available, modified_on, modified_by, action)
SELECT 
  p.id_product, 
  p.name, 
  p.description, 
  p.price, 
  p.available, 
  p.created_on, 
  (SELECT id_user FROM users WHERE email = 'admin@example.com'), 
  'update'
FROM products p
WHERE p.name IN ('Brownie Clásico', 'Suspiros');