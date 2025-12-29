// Trust types
export type TrustState = 'probation' | 'learning' | 'trusted' | 'verified' | 'restricted';
export type ActionMode = 'suggest' | 'supervised' | 'autonomous' | 'full_auto';

export interface TrustFactors {
  accuracy: number;
  compliance: number;
  calibration: number;
  recency: number;
  reversals: number;
}

export interface TrustScore {
  domain: string;
  value: number;
  state: TrustState;
  factors: TrustFactors;
  action_count: number;
  last_activity: string;
  state_entered: string;
}

export interface TrustOverview {
  overall_score: number;
  overall_state: TrustState;
  domain_count: number;
  interpretation: string;
}

// Hat types
export interface Hat {
  id: string;
  name: string;
  description: string;
  color: string;
  icon: string;
  is_active: boolean;
}

// Item types
export type ItemStatus = 'pending' | 'in_progress' | 'done' | 'archived';
export type ItemType = 'email' | 'task' | 'event' | 'document' | 'note';

export interface Item {
  id: string;
  type: ItemType;
  hat_id: string;
  from: string;
  subject: string;
  body: string;
  priority: number;
  status: ItemStatus;
  created_at: string;
  updated_at: string;
}

// Stats types
export interface Stats {
  identity: string;
  total_items: number;
  total_memories: number;
  total_spaces: number;
  items_by_hat: Record<string, number>;
  agent_running: boolean;
  trust?: {
    overall_score: number;
    domain_count: number;
    domains: Record<string, { value: number; state: TrustState }>;
  };
}

// Ledger types
export interface LedgerEntry {
  id: string;
  sequence: number;
  timestamp: string;
  action: string;
  actor: string;
  entity_type: string;
  entity_id: string;
  details: Record<string, unknown>;
  hash: string;
  prev_hash: string;
}

export interface LedgerSummary {
  total_entries: number;
  first_entry: string;
  last_entry: string;
  chain_valid: boolean;
  actions_by_type: Record<string, number>;
  actors_summary: Record<string, number>;
}
