-- Migration: Add is_active column to users table
-- Created: 2024-06-17
-- Description: Add user activation/deactivation functionality

-- Add is_active column to users table
ALTER TABLE users 
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Create index for better performance when filtering by active users
CREATE INDEX idx_users_is_active ON users(is_active);

-- Update existing users to be active by default (this is redundant with DEFAULT TRUE, but explicit)
UPDATE users SET is_active = TRUE WHERE is_active IS NULL;

-- Add comment to the column for documentation
COMMENT ON COLUMN users.is_active IS 'Indicates whether the user account is active and can authenticate';