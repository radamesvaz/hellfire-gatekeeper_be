-- Seed superadmin user (idempotent).
-- Added in a follow-up migration because 000040 was already applied in some
-- environments before the user INSERT was added to that file.
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
