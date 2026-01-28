-- Migration: Create industries table
-- Description: Add industries table to support 20 industries organized in categories
-- Date: 2026-01-27

BEGIN;

-- Create industries table
CREATE TABLE IF NOT EXISTS industries (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    icon VARCHAR(50),
    osm_primary_tag VARCHAR(100) NOT NULL,
    osm_additional_tags TEXT[],
    description TEXT,
    active BOOLEAN DEFAULT true NOT NULL,
    sort_order INTEGER DEFAULT 0 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_industries_category ON industries(category);
CREATE INDEX IF NOT EXISTS idx_industries_active ON industries(active);
CREATE INDEX IF NOT EXISTS idx_industries_sort_order ON industries(sort_order);

-- Create updated_at trigger
CREATE OR REPLACE FUNCTION update_industries_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_industries_updated_at
    BEFORE UPDATE ON industries
    FOR EACH ROW
    EXECUTE FUNCTION update_industries_updated_at();

-- Seed industries data
INSERT INTO industries (id, name, category, icon, osm_primary_tag, osm_additional_tags, description, active, sort_order) VALUES
-- Personal Care & Beauty
('tattoo', 'Tattoo Studios', 'personal_care', '🎨', 'shop=tattoo', '{}', 'Tattoo and body art studios', true, 1),
('beauty', 'Beauty Salons', 'personal_care', '💅', 'shop=beauty', '{}', 'Beauty salons and cosmetic services', true, 2),
('barber', 'Barber Shops', 'personal_care', '💈', 'shop=hairdresser', '{"shop=barber"}', 'Barbershops and hair salons', true, 3),
('spa', 'Spas & Wellness', 'personal_care', '🧖', 'leisure=spa', '{"amenity=spa"}', 'Spas and wellness centers', true, 4),
('nail_salon', 'Nail Salons', 'personal_care', '💅', 'shop=beauty', '{"beauty=nails"}', 'Nail salons and manicure services', true, 5),

-- Health & Wellness
('gym', 'Gyms & Fitness', 'health_wellness', '💪', 'leisure=fitness_centre', '{"leisure=sports_centre", "amenity=gym"}', 'Gyms and fitness centers', true, 6),
('dentist', 'Dentists', 'health_wellness', '🦷', 'amenity=dentist', '{}', 'Dental clinics and dentists', true, 7),
('pharmacy', 'Pharmacies', 'health_wellness', '💊', 'amenity=pharmacy', '{}', 'Pharmacies and drugstores', true, 8),
('massage', 'Massage Therapy', 'health_wellness', '💆', 'shop=massage', '{"amenity=massage"}', 'Massage therapy and wellness centers', true, 9),

-- Food & Beverage
('restaurant', 'Restaurants', 'food_beverage', '🍽️', 'amenity=restaurant', '{}', 'Restaurants and dining establishments', true, 10),
('cafe', 'Cafes & Coffee Shops', 'food_beverage', '☕', 'amenity=cafe', '{}', 'Cafes and coffee shops', true, 11),
('bar', 'Bars & Pubs', 'food_beverage', '🍺', 'amenity=bar', '{"amenity=pub"}', 'Bars, pubs, and nightlife venues', true, 12),
('bakery', 'Bakeries', 'food_beverage', '🥖', 'shop=bakery', '{}', 'Bakeries and pastry shops', true, 13),

-- Automotive
('car_repair', 'Car Repair', 'automotive', '🔧', 'shop=car_repair', '{}', 'Auto repair and maintenance shops', true, 14),
('car_wash', 'Car Wash', 'automotive', '🚗', 'amenity=car_wash', '{}', 'Car wash and detailing services', true, 15),
('car_dealer', 'Car Dealers', 'automotive', '🚙', 'shop=car', '{}', 'Car dealerships and sales', true, 16),

-- Retail
('clothing', 'Clothing Stores', 'retail', '👕', 'shop=clothes', '{}', 'Clothing and fashion stores', true, 17),
('convenience', 'Convenience Stores', 'retail', '🏪', 'shop=convenience', '{}', 'Convenience stores and mini markets', true, 18),

-- Professional Services
('lawyer', 'Lawyers', 'professional', '⚖️', 'office=lawyer', '{}', 'Law offices and legal services', true, 19),
('accountant', 'Accountants', 'professional', '📊', 'office=accountant', '{}', 'Accounting and financial services', true, 20)
ON CONFLICT (id) DO NOTHING;

COMMIT;
