CREATE TABLE users (
    id_user INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    password_hash VARCHAR(255),
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id_product INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    available BOOLEAN DEFAULT TRUE,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products_history (
    id_products_history INT AUTO_INCREMENT PRIMARY KEY,
    id_product INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    available BOOLEAN,
    modified_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_by VARCHAR(100),
    action ENUM('update', 'delete') NOT NULL,
    FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE CASCADE
);

CREATE TABLE orders (
    id_order INT AUTO_INCREMENT PRIMARY KEY,
    id_user INT NOT NULL,
    status ENUM('pending', 'preparing', 'ready', 'delivered', 'cancelled') DEFAULT 'pending',
    total_price DECIMAL(10,2) NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE CASCADE
);

CREATE TABLE orders_history (
    id_order_history INT AUTO_INCREMENT PRIMARY KEY,
    id_order INT NOT NULL,
    id_user INT NOT NULL,
    status ENUM('pending', 'preparing', 'ready', 'delivered', 'cancelled') NOT NULL,
    total_price DECIMAL(10,2) NOT NULL,
    modified_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_by VARCHAR(100),
    action ENUM('update', 'delete') NOT NULL,
    FOREIGN KEY (id_order) REFERENCES orders(id_order) ON DELETE CASCADE,
    FOREIGN KEY (id_user) REFERENCES users(id_user) ON DELETE CASCADE
);

CREATE TABLE order_items (
    id_order_items INT NOT NULL,
    id_product INT NOT NULL,
    quantity INT NOT NULL,
    PRIMARY KEY (id_order_items, id_product),
    FOREIGN KEY (id_order_items) REFERENCES orders(id_order) ON DELETE CASCADE,
    FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE CASCADE
);
