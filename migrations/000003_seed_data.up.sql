DELETE FROM products_history WHERE id_product IN (1, 2);
DELETE FROM products WHERE id_product IN (1, 2);
DELETE FROM users WHERE id_user IN (1, 2);
DELETE FROM roles WHERE id_role IN (1, 2);

INSERT INTO roles (id_role, name) VALUES
  (1, 'admin'),
  (2, 'client');

INSERT INTO users (id_user, id_role, name, email, phone, password_hash, created_on)
VALUES (
  1,
  1,
  'Admin User',
  'admin@example.com',
  '55-55555',
  '$2a$10$SO6GoHITlrEuH9mZyaFqN.vmtx9F0mn.4c2Blp7xM9WCM1svOMnYq',
  '2025-04-14 10:00:00'
),
(
  2,
  2,
  'Client',
  'client@example.com',
  '66-6666',
  NULL,
  '2025-04-14 10:00:00'
);

INSERT INTO products (id_product, name, description, price, available, created_on)
VALUES
  (1, 'Brownie Clásico', 'Delicioso brownie de chocolate', 3.5, true, '2025-04-14 10:00:00'),
  (2, 'Suspiros', 'Suspiros tradicionales', 5, true, '2025-04-14 10:00:00');

INSERT INTO products_history (id_products_history, id_product, name, description, price, available, modified_on, modified_by, action)
VALUES
  (1, 1, 'Brownie Clásico', 'Delicioso brownie de chocolate', 3.5, true, '2025-04-14 10:00:00', 1, 'update'),
  (2, 2, 'Suspiros', 'Suspiros tradicionales', 5, true, '2025-04-14 10:00:00', 1, 'update');