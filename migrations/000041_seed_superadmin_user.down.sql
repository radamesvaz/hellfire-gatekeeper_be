DELETE FROM users
WHERE tenant_id = 1
  AND email = 'superadmin@example.com'
  AND id_role = 3;
