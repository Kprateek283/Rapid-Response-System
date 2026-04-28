-- guests: Active guest sessions
CREATE TABLE IF NOT EXISTS guests (
    id VARCHAR(50) PRIMARY KEY,  -- "g_{random_hex}"
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    hotel_id UUID REFERENCES hotels(id) ON DELETE CASCADE,
    guest_name VARCHAR(255) NOT NULL,
    device_fingerprint VARCHAR(255),
    expected_checkout TIMESTAMPTZ NOT NULL,
    ble_paired BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- incidents: Core incident lifecycle
CREATE TABLE IF NOT EXISTS incidents (
    id VARCHAR(50) PRIMARY KEY,  -- "inc_{random_hex}"
    hotel_id UUID REFERENCES hotels(id),
    room_id UUID REFERENCES rooms(id),
    guest_id VARCHAR(50) REFERENCES guests(id),
    trigger_source VARCHAR(50) NOT NULL,  -- MANUAL_PHONE, BLE_BUTTON, VENUE_SENSOR
    gps_lat DOUBLE PRECISION,
    gps_lng DOUBLE PRECISION,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    -- OPEN, PROCESSING, DISPATCHED, ARRIVED, ASSESSED, STABILIZING, CONTROLLED, RESOLVED, CLOSED
    initial_severity INT,
    ground_truth_severity INT,
    final_severity INT,
    ai_summary TEXT,
    media_video_url TEXT,
    media_audio_stream_id VARCHAR(100),
    escalation_status VARCHAR(50),
    resolution_proposed_by VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ
);

-- dispatch_assignments: Staff-to-incident mapping
CREATE TABLE IF NOT EXISTS dispatch_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id VARCHAR(50) REFERENCES incidents(id),
    staff_id UUID REFERENCES staff(id),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    -- PENDING, NOTIFIED, ACCEPTED, DECLINED, IN_TRANSIT, ARRIVED, ON_SITE_EXPENDABLE, RELEASED, TIMED_OUT
    accepted_at TIMESTAMPTZ,
    arrived_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    decline_reason VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- incident_events: Immutable audit log
CREATE TABLE IF NOT EXISTS incident_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id VARCHAR(50) REFERENCES incidents(id),
    event_type VARCHAR(100) NOT NULL,
    actor_id VARCHAR(100),
    actor_role VARCHAR(50),
    payload JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- incident_reports: Final immutable reports
CREATE TABLE IF NOT EXISTS incident_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id VARCHAR(50) UNIQUE REFERENCES incidents(id),
    report_json JSONB NOT NULL,
    compiled_by VARCHAR(50) DEFAULT 'AI',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_incidents_hotel ON incidents(hotel_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_dispatch_incident ON dispatch_assignments(incident_id);
CREATE INDEX IF NOT EXISTS idx_dispatch_staff ON dispatch_assignments(staff_id);
CREATE INDEX IF NOT EXISTS idx_events_incident ON incident_events(incident_id);
CREATE INDEX IF NOT EXISTS idx_guests_room ON guests(room_id);
