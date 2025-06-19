-- Migración inicial para la base de datos ExactoGas
-- Basado en el documento "06_Diseño_de_Base_de_Datos.md"

-- Crear tipos enumerados
CREATE TYPE user_role_enum AS ENUM ('CLIENT', 'REPARTIDOR', 'ADMIN');
CREATE TYPE order_status_enum AS ENUM ('PENDING', 'PENDING_OUT_OF_HOURS', 'CONFIRMED', 'ASSIGNED', 'IN_TRANSIT', 'DELIVERED', 'CANCELLED');

-- Tabla de usuarios
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) NOT NULL UNIQUE,
    user_role user_role_enum NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabla de productos
CREATE TABLE products (
    product_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL CHECK (price > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabla de pedidos
CREATE TABLE orders (
    order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL CHECK (total_amount >= 0),
    latitude NUMERIC(9, 6) NOT NULL,
    longitude NUMERIC(9, 6) NOT NULL,
    delivery_address_text TEXT NOT NULL,
    payment_note VARCHAR(255),
    order_status order_status_enum NOT NULL,
    order_time TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    estimated_arrival_time TIMESTAMPTZ,
    assigned_repartidor_id UUID,
    assigned_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_client FOREIGN KEY (client_id) REFERENCES users(user_id) ON DELETE RESTRICT,
    CONSTRAINT fk_assigned_repartidor FOREIGN KEY (assigned_repartidor_id) REFERENCES users(user_id) ON DELETE SET NULL
);

-- Tabla de items de pedido
CREATE TABLE order_items (
    order_item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL CHECK (unit_price > 0),
    subtotal DECIMAL(10, 2) NOT NULL CHECK (subtotal >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE,
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(product_id) ON DELETE RESTRICT,
    CONSTRAINT uq_order_items_order_product UNIQUE (order_id, product_id)
);

-- Índices
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_phone_number ON users (phone_number);
CREATE INDEX idx_users_user_role ON users (user_role);

CREATE INDEX idx_products_name ON products (name);
CREATE INDEX idx_products_is_active ON products (is_active);

CREATE INDEX idx_orders_client_id ON orders (client_id);
CREATE INDEX idx_orders_status ON orders (order_status);
CREATE INDEX idx_orders_assigned_repartidor_id ON orders (assigned_repartidor_id);
CREATE INDEX idx_orders_assigned_at ON orders (assigned_at);
CREATE INDEX idx_orders_order_time ON orders (order_time DESC);
CREATE INDEX idx_orders_location ON orders (latitude, longitude);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_product_id ON order_items (product_id);

-- Función para actualizar el total del pedido
CREATE OR REPLACE FUNCTION update_order_total_amount()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE orders
    SET total_amount = (
        SELECT COALESCE(SUM(subtotal), 0)
        FROM order_items
        WHERE order_id = NEW.order_id
    )
    WHERE order_id = NEW.order_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para actualizar el total del pedido
CREATE TRIGGER trigger_update_order_total_amount_on_item_change
AFTER INSERT OR UPDATE OR DELETE ON order_items
FOR EACH ROW
EXECUTE FUNCTION update_order_total_amount();

-- Función para actualizar automáticamente updated_at
CREATE OR REPLACE FUNCTION set_updated_at_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Aplicar trigger de updated_at a todas las tablas
CREATE TRIGGER trigger_set_updated_at_users
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

CREATE TRIGGER trigger_set_updated_at_products
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

CREATE TRIGGER trigger_set_updated_at_orders
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

CREATE TRIGGER trigger_set_updated_at_order_items
BEFORE UPDATE ON order_items
FOR EACH ROW
EXECUTE FUNCTION set_updated_at_timestamp();

-- Insertar datos de ejemplo para productos
INSERT INTO products (name, description, price, is_active) VALUES
('Balón de Gas 10kg', 'Balón de gas doméstico de 10kg', 45.00, TRUE),
('Balón de Gas 15kg', 'Balón de gas doméstico de 15kg', 65.00, TRUE),
('Balón de Gas 30kg', 'Balón de gas industrial de 30kg', 120.00, TRUE);

-- Insertar usuario administrador por defecto
INSERT INTO users (email, password_hash, full_name, phone_number, user_role) VALUES
('admin@exactogas.com', '$2a$10$pTBwStnVRXfeS5R0I9j12.AiGLAwMBDpoQZ0r5qCA198wn1LtpX2W', 'Administrador', '999888777', 'ADMIN');
-- La contraseña hasheada es "admin123"

-- Vista de productos activos
CREATE OR REPLACE VIEW view_active_products AS
SELECT
    product_id,
    name,
    description,
    price
FROM products
WHERE is_active = TRUE;

-- Vista de pedidos con información del cliente y repartidor
CREATE OR REPLACE VIEW view_client_orders AS
SELECT
    o.order_id,
    o.total_amount,
    o.delivery_address_text,
    o.order_status,
    o.order_time,
    o.estimated_arrival_time,
    o.delivered_at,
    o.cancelled_at,
    o.created_at,
    c.full_name AS client_name,
    c.phone_number AS client_phone,
    r.full_name AS repartidor_name,
    r.phone_number AS repartidor_phone
FROM orders o
JOIN users c ON o.client_id = c.user_id
LEFT JOIN users r ON o.assigned_repartidor_id = r.user_id;

-- Vista de pedidos pendientes para repartidores
CREATE OR REPLACE VIEW view_repartidor_pending_orders AS
SELECT
    o.order_id,
    o.total_amount,
    o.latitude,
    o.longitude,
    o.delivery_address_text,
    o.order_status,
    o.order_time,
    o.estimated_arrival_time,
    c.full_name AS client_name,
    c.phone_number AS client_phone
FROM orders o
JOIN users c ON o.client_id = c.user_id
WHERE o.order_status IN ('PENDING', 'CONFIRMED', 'ASSIGNED', 'IN_TRANSIT')
ORDER BY o.order_time ASC; 