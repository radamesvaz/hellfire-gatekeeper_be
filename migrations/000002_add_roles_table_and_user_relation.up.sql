-- Creating new roles table
CREATE TABLE roles (
  id_role INT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(50) NOT NULL UNIQUE
);

-- Adding values to the roles table
INSERT INTO roles (name) VALUES ('admin'), ('client');

-- Modify the users table to add a id_role column
ALTER TABLE users ADD COLUMN id_role INT NOT NULL DEFAULT 2 AFTER id_user,
ADD CONSTRAINT fk_role FOREIGN KEY (id_role) REFERENCES roles(id_role);