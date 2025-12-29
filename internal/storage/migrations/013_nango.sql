-- Nango integration: Track auth source for each space
--
-- ARCHITECTURAL PRINCIPLE:
-- Auth infrastructure (Nango) is separate from authorization/agency (QuantumLife).
-- OAuth/token possession does NOT imply permission-to-act.
-- The autonomy modes (Suggest/Supervised/Autonomous) enforce this boundary.

-- Auth source tells us where to fetch credentials
-- - 'custom': Use QuantumLife's credential store (current behavior)
-- - 'nango': Use Nango API to get credentials
-- This allows gradual migration: existing connections stay 'custom',
-- new connections can use 'nango'
ALTER TABLE spaces ADD COLUMN auth_source TEXT DEFAULT 'custom';

-- Nango connection ID (maps to Nango's connection_id)
-- For spaces with auth_source='nango', this is the connection identifier
ALTER TABLE spaces ADD COLUMN nango_connection_id TEXT;

-- Create index for looking up by nango connection
CREATE INDEX IF NOT EXISTS idx_spaces_nango_connection ON spaces(nango_connection_id);
