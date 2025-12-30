-- Add youtube_cookie column to profiles table for storing YouTube authentication cookies
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS youtube_cookie TEXT;
