import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import type { Stats, Hat, Item, TrustScore, TrustOverview, LedgerEntry, LedgerSummary } from '../types';

// Stats
export function useStats() {
  return useQuery({
    queryKey: ['stats'],
    queryFn: () => api.get<Stats>('/stats'),
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

// Hats
export function useHats() {
  return useQuery({
    queryKey: ['hats'],
    queryFn: () => api.get<Hat[]>('/hats'),
  });
}

export function useHat(hatId: string) {
  return useQuery({
    queryKey: ['hats', hatId],
    queryFn: () => api.get<Hat>(`/hats/${hatId}`),
    enabled: !!hatId,
  });
}

// Items
export function useItems(hatId?: string) {
  return useQuery({
    queryKey: ['items', hatId],
    queryFn: () => api.get<Item[]>(hatId ? `/items?hat=${hatId}` : '/items'),
  });
}

export function useItem(itemId: string) {
  return useQuery({
    queryKey: ['items', itemId],
    queryFn: () => api.get<Item>(`/items/${itemId}`),
    enabled: !!itemId,
  });
}

export function useUpdateItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ itemId, updates }: { itemId: string; updates: Partial<Item> }) =>
      api.put<Item>(`/items/${itemId}`, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['items'] });
    },
  });
}

// Trust
export function useTrustScores() {
  return useQuery({
    queryKey: ['trust', 'scores'],
    queryFn: () => api.get<{ scores: TrustScore[]; count: number }>('/trust/'),
  });
}

export function useTrustOverall() {
  return useQuery({
    queryKey: ['trust', 'overall'],
    queryFn: () => api.get<TrustOverview>('/trust/overall'),
  });
}

export function useTrustDomain(domain: string) {
  return useQuery({
    queryKey: ['trust', 'domain', domain],
    queryFn: () => api.get<TrustScore & { interpretation: string }>(`/trust/domain/${domain}`),
    enabled: !!domain,
  });
}

export function useAutonomyLevel(domain: string, confidence?: number) {
  return useQuery({
    queryKey: ['trust', 'autonomy', domain, confidence],
    queryFn: () => api.get<{
      domain: string;
      confidence: number;
      mode: string;
      description: string;
      trust_value: number;
      trust_state: string;
    }>(`/trust/domain/${domain}/autonomy${confidence ? `?confidence=${confidence}` : ''}`),
    enabled: !!domain,
  });
}

// Ledger
export function useLedgerEntries(limit = 50, offset = 0) {
  return useQuery({
    queryKey: ['ledger', 'entries', limit, offset],
    queryFn: () => api.get<{ entries: LedgerEntry[]; total: number; limit: number; offset: number }>(
      `/ledger?limit=${limit}&offset=${offset}`
    ),
  });
}

export function useLedgerSummary() {
  return useQuery({
    queryKey: ['ledger', 'summary'],
    queryFn: () => api.get<LedgerSummary>('/ledger/summary'),
  });
}

export function useLedgerVerify() {
  return useQuery({
    queryKey: ['ledger', 'verify'],
    queryFn: () => api.get<{ valid: boolean; entries_checked: number; error?: string }>('/ledger/verify'),
    refetchInterval: 60000, // Verify every minute
  });
}
