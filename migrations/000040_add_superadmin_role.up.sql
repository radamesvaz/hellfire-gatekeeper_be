INSERT INTO roles (id_role, name) VALUES (3, 'superadmin');

SELECT setval(pg_get_serial_sequence('roles', 'id_role'), (SELECT MAX(id_role) FROM roles));
