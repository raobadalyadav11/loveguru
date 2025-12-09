-- Add device token support for push notifications
ALTER TABLE users ADD COLUMN IF NOT EXISTS fcm_token TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS apns_token TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_type TEXT CHECK (device_type IN ('IOS', 'ANDROID', 'WEB'));

-- Create index for faster device token lookups
CREATE INDEX IF NOT EXISTS idx_users_fcm_token ON users(fcm_token);
CREATE INDEX IF NOT EXISTS idx_users_apns_token ON users(apns_token);

-- Add FAQ table for AI assistant
CREATE TABLE IF NOT EXISTS faqs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    category TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add call status tracking
ALTER TABLE call_logs ADD COLUMN IF NOT EXISTS status_update TEXT;
ALTER TABLE call_logs ADD COLUMN IF NOT EXISTS status_timestamp TIMESTAMPTZ DEFAULT NOW();

-- Add call feedback prompts
CREATE TABLE IF NOT EXISTS call_feedback_prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    advisor_id UUID NOT NULL REFERENCES users(id),
    prompt_sent_at TIMESTAMPTZ DEFAULT NOW(),
    response_received_at TIMESTAMPTZ,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    feedback_text TEXT
);

-- Add specializations management for admin
CREATE TABLE IF NOT EXISTS specializations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    category TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default specializations
INSERT INTO specializations (name, description, category) VALUES
('Relationship Guidance', 'Help with relationship issues and communication', 'Counseling'),
('Dating Advice', 'Tips and strategies for dating and meeting people', 'Dating'),
('Breakup Recovery', 'Support and guidance for getting over breakups', 'Counseling'),
('Marriage Counseling', 'Advice for married couples and long-term relationships', 'Counseling'),
('Confidence Building', 'Help building self-confidence and self-esteem', 'Personal Development'),
('Long Distance Relationships', 'Guidance for maintaining long-distance relationships', 'Counseling'),
('LGBTQ+ Support', 'Specialized support for LGBTQ+ relationships', 'Specialized'),
('Online Dating', 'Advice on online dating platforms and apps', 'Dating')
ON CONFLICT (name) DO NOTHING;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_faqs_category ON faqs(category);
CREATE INDEX IF NOT EXISTS idx_faqs_active ON faqs(is_active);
CREATE INDEX IF NOT EXISTS idx_call_feedback_session ON call_feedback_prompts(session_id);
CREATE INDEX IF NOT EXISTS idx_specializations_category ON specializations(category);