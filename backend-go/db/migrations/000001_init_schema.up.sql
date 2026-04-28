-- Enable PostGIS extension for spatial queries
CREATE EXTENSION IF NOT EXISTS postgis;

-- 1. Multi-Tenant Group
CREATE TABLE hotel_groups (
                              id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                              name VARCHAR(255) NOT NULL,
                              contact_email VARCHAR(255) UNIQUE,
                              created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Physical Properties
CREATE TABLE hotels (
                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        group_id UUID REFERENCES hotel_groups(id) ON DELETE CASCADE,
                        name VARCHAR(255) NOT NULL,
                        address TEXT,
                        coordinates GEOMETRY(Point, 4326),
                        timezone VARCHAR(50) DEFAULT 'UTC',
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. Hotel Rooms
CREATE TABLE rooms (
                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       hotel_id UUID REFERENCES hotels(id) ON DELETE CASCADE,
                       room_number VARCHAR(50) NOT NULL,
                       floor_level INT NOT NULL,
                       qr_index INT UNIQUE NOT NULL,
                       is_occupied BOOLEAN DEFAULT false,
                       UNIQUE(hotel_id, room_number)
);

-- 4. Static Staff Data
CREATE TABLE staff (
                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       hotel_id UUID REFERENCES hotels(id) ON DELETE CASCADE,
                       name VARCHAR(255) NOT NULL,
                       phone_number VARCHAR(20) NOT NULL,
                       role VARCHAR(50) NOT NULL,
                       skill_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                       UNIQUE(hotel_id, phone_number)
);

--5. Users Table
CREATE TABLE users (
                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       email VARCHAR(255) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL,
                       name VARCHAR(255) NOT NULL,
                       role VARCHAR(50) NOT NULL,
                       group_id UUID REFERENCES hotel_groups(id) ON DELETE CASCADE,
                       hotel_id UUID REFERENCES hotels(id) ON DELETE CASCADE,
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);