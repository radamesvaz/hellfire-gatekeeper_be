-- Remove all users with the superadmin role before deleting the role,
-- otherwise users.id_role FK (fk_role) blocks the DELETE.
DELETE FROM users WHERE id_role = 3;
DELETE FROM roles WHERE id_role = 3 AND name = 'superadmin';
