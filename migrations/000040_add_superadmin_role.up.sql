INSERT INTO roles (id_role, name) VALUES (3, 'superadmin');

SELECT setval(pg_get_serial_sequence('roles', 'id_role'), (SELECT MAX(id_role) FROM roles));

-- Platform operator for internal APIs (tenant signup codes, etc.).
-- Password hash matches seeded adminpass from 000003 (bcrypt cost 10).
INSERT INTO users (tenant_id, id_role, name, email, phone, password_hash, created_on)
SELECT
  1,
  3,
  'Super Admin',
  'superadmin@example.com',
  '00-00000',
  '$2a$10$SO6GoHITlrEuH9mZyaFqN.vmtx9F0mn.4c2Blp7xM9WCM1svOMnYq',
  NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM users WHERE tenant_id = 1 AND email = 'superadmin@example.com'
);
