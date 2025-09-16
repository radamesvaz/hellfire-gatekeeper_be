-- Update existing mock data to explicitly set paid status
UPDATE orders SET paid = FALSE WHERE id_order IN (1, 2, 3, 4);
