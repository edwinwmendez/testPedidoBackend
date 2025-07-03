-- Migration: Add product analytics and ratings support
-- File: 005_add_product_analytics.sql

-- Add analytics columns to products table
ALTER TABLE products 
ADD COLUMN view_count INTEGER DEFAULT 0 NOT NULL,
ADD COLUMN purchase_count INTEGER DEFAULT 0 NOT NULL,
ADD COLUMN rating_average DECIMAL(3,2) DEFAULT 0.00 NOT NULL,
ADD COLUMN rating_count INTEGER DEFAULT 0 NOT NULL,
ADD COLUMN popularity_score DECIMAL(10,2) DEFAULT 0.00 NOT NULL;

-- Create product_ratings table for customer ratings
CREATE TABLE product_ratings (
    rating_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure one rating per user per product
    UNIQUE(product_id, user_id)
);

-- Add indexes for performance
CREATE INDEX idx_products_popularity_score ON products(popularity_score DESC);
CREATE INDEX idx_products_created_at ON products(created_at DESC);
CREATE INDEX idx_products_view_count ON products(view_count DESC);
CREATE INDEX idx_products_purchase_count ON products(purchase_count DESC);
CREATE INDEX idx_product_ratings_product_id ON product_ratings(product_id);
CREATE INDEX idx_product_ratings_user_id ON product_ratings(user_id);

-- Function to update product rating average
CREATE OR REPLACE FUNCTION update_product_rating_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update rating statistics for the product
    UPDATE products 
    SET 
        rating_average = (
            SELECT COALESCE(AVG(rating), 0)
            FROM product_ratings 
            WHERE product_id = COALESCE(NEW.product_id, OLD.product_id)
        ),
        rating_count = (
            SELECT COUNT(*)
            FROM product_ratings 
            WHERE product_id = COALESCE(NEW.product_id, OLD.product_id)
        ),
        updated_at = CURRENT_TIMESTAMP
    WHERE product_id = COALESCE(NEW.product_id, OLD.product_id);
    
    -- Update popularity score
    UPDATE products 
    SET popularity_score = (
        (purchase_count * 3.0) + 
        (view_count * 0.1) + 
        (rating_average * 2.0) + 
        (rating_count * 1.0)
    )
    WHERE product_id = COALESCE(NEW.product_id, OLD.product_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create triggers for automatic rating updates
CREATE TRIGGER trigger_update_product_rating_stats
    AFTER INSERT OR UPDATE OR DELETE ON product_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_product_rating_stats();

-- Function to update popularity score when view/purchase counts change
CREATE OR REPLACE FUNCTION update_popularity_score()
RETURNS TRIGGER AS $$
BEGIN
    NEW.popularity_score = (
        (NEW.purchase_count * 3.0) + 
        (NEW.view_count * 0.1) + 
        (NEW.rating_average * 2.0) + 
        (NEW.rating_count * 1.0)
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for popularity score updates
CREATE TRIGGER trigger_update_popularity_score
    BEFORE UPDATE OF view_count, purchase_count, rating_average, rating_count ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_popularity_score();