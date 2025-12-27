# QuantumLife API Reference

**REST + WebSocket API**

---

## Overview

QuantumLife exposes a local API for UI clients and integrations. The API runs on `localhost:8420` by default.

**Base URL:** `http://localhost:8420/api/v1`

**Authentication:** Bearer token (generated during identity creation)

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8420/api/v1/me
```

## Authentication

### Generate Token

```http
POST /api/v1/auth/token
Content-Type: application/json

{
  "password": "your-master-password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJFZDI1NTE5...",
  "expires_at": "2025-01-15T10:00:00Z"
}
```

### Verify Token

```http
GET /api/v1/auth/verify
Authorization: Bearer <token>
```

**Response:**
```json
{
  "valid": true,
  "identity_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2025-01-15T10:00:00Z"
}
```

---

## Identity

### Get Identity (YOU)

```http
GET /api/v1/me
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "display_name": "Alice",
  "did": "did:key:z6Mk...",
  "created_at": "2025-01-01T00:00:00Z",
  "public_keys": {
    "signing": "z6Mk...",
    "encryption": "z6LS...",
    "pq_signing": "z6Mq...",
    "pq_encryption": "z6Lr..."
  }
}
```

### Update Identity

```http
PUT /api/v1/me
Content-Type: application/json

{
  "display_name": "Alice Smith"
}
```

---

## Hats

### List Hats

```http
GET /api/v1/hats
```

**Query Parameters:**
- `include_stats` (boolean) - Include item counts

**Response:**
```json
{
  "hats": [
    {
      "id": "hat-001",
      "name": "Parent",
      "description": "School, activities, kids' health",
      "icon": "family",
      "color": "#FF6B6B",
      "priority": 1,
      "is_default": false,
      "item_count": 47,
      "unread_count": 3,
      "config": {
        "notify_urgent": true,
        "notify_normal": false,
        "auto_archive": false
      }
    }
  ]
}
```

### Create Hat

```http
POST /api/v1/hats
Content-Type: application/json

{
  "name": "Side Project",
  "description": "My weekend startup",
  "icon": "rocket",
  "color": "#9B59B6",
  "config": {
    "notify_urgent": true,
    "notify_normal": true,
    "importance_floor": 0.3
  }
}
```

**Response:**
```json
{
  "id": "hat-013",
  "name": "Side Project",
  ...
}
```

### Get Hat

```http
GET /api/v1/hats/{id}
```

### Update Hat

```http
PUT /api/v1/hats/{id}
Content-Type: application/json

{
  "name": "Startup",
  "priority": 2
}
```

### Delete Hat

```http
DELETE /api/v1/hats/{id}
```

**Note:** Items in this hat will be moved to the default Inbox hat.

### List Hat Items

```http
GET /api/v1/hats/{id}/items
```

**Query Parameters:**
- `limit` (int) - Max items to return (default: 50)
- `offset` (int) - Pagination offset
- `type` (string) - Filter by item type
- `requires_action` (boolean) - Filter actionable items
- `since` (timestamp) - Items after this time

**Response:**
```json
{
  "items": [...],
  "total": 47,
  "has_more": true
}
```

---

## Spaces

### List Spaces

```http
GET /api/v1/spaces
```

**Response:**
```json
{
  "spaces": [
    {
      "id": "space-001",
      "type": "gmail",
      "name": "Personal Gmail",
      "status": "active",
      "last_sync": "2025-01-15T09:30:00Z",
      "item_count": 1250
    },
    {
      "id": "space-002",
      "type": "calendar",
      "name": "Google Calendar",
      "status": "active",
      "last_sync": "2025-01-15T09:30:00Z",
      "item_count": 89
    }
  ]
}
```

### Connect Space

```http
POST /api/v1/spaces
Content-Type: application/json

{
  "type": "gmail",
  "name": "Work Gmail",
  "config": {
    "oauth_code": "4/0AX4XfWi..."
  }
}
```

**Response:**
```json
{
  "id": "space-003",
  "type": "gmail",
  "name": "Work Gmail",
  "status": "connecting",
  "oauth_url": null
}
```

### Get Space

```http
GET /api/v1/spaces/{id}
```

### Start OAuth Flow

For spaces requiring OAuth:

```http
POST /api/v1/spaces/oauth/start
Content-Type: application/json

{
  "type": "gmail"
}
```

**Response:**
```json
{
  "oauth_url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "state": "random-state-token"
}
```

### Complete OAuth Flow

```http
POST /api/v1/spaces/oauth/callback
Content-Type: application/json

{
  "code": "4/0AX4XfWi...",
  "state": "random-state-token"
}
```

### Disconnect Space

```http
DELETE /api/v1/spaces/{id}
```

### Trigger Sync

```http
POST /api/v1/spaces/{id}/sync
```

**Response:**
```json
{
  "status": "syncing",
  "started_at": "2025-01-15T10:00:00Z"
}
```

### Get Sync Status

```http
GET /api/v1/spaces/{id}/sync
```

**Response:**
```json
{
  "status": "completed",
  "started_at": "2025-01-15T10:00:00Z",
  "completed_at": "2025-01-15T10:00:05Z",
  "items_synced": 15,
  "errors": []
}
```

---

## Items

### List Items

```http
GET /api/v1/items
```

**Query Parameters:**
- `hat_id` (uuid) - Filter by hat
- `space_id` (uuid) - Filter by space
- `type` (string) - Filter by type (email, event, document, etc.)
- `requires_action` (boolean) - Filter actionable items
- `importance_min` (float) - Minimum importance (0.0-1.0)
- `since` (timestamp) - Items after this time
- `limit` (int) - Max items (default: 50)
- `offset` (int) - Pagination offset

**Response:**
```json
{
  "items": [
    {
      "id": "item-001",
      "space_id": "space-001",
      "hat_id": "hat-001",
      "type": "email",
      "external_id": "gmail:abc123",
      "metadata": {
        "from": "school@example.com",
        "subject": "Parent-Teacher Conference",
        "date": "2025-01-15T08:00:00Z"
      },
      "importance": 0.85,
      "requires_action": true,
      "action_deadline": "2025-01-20T00:00:00Z",
      "created_at": "2025-01-15T08:00:00Z"
    }
  ],
  "total": 1250,
  "has_more": true
}
```

### Get Item

```http
GET /api/v1/items/{id}
```

**Response includes full content:**
```json
{
  "id": "item-001",
  ...
  "content": {
    "body": "Dear Parent, We would like to invite you...",
    "html": "<html>...",
    "attachments": [
      {
        "name": "schedule.pdf",
        "type": "application/pdf",
        "size": 45678
      }
    ]
  }
}
```

### Update Item

```http
PUT /api/v1/items/{id}
Content-Type: application/json

{
  "hat_id": "hat-002",
  "requires_action": false
}
```

### Search Items

```http
POST /api/v1/items/search
Content-Type: application/json

{
  "query": "budget meeting Q4",
  "filters": {
    "hat_ids": ["hat-001", "hat-002"],
    "types": ["email", "document"],
    "date_range": {
      "start": "2024-10-01T00:00:00Z",
      "end": "2024-12-31T23:59:59Z"
    }
  },
  "limit": 20
}
```

**Response:**
```json
{
  "results": [
    {
      "item": {...},
      "score": 0.92,
      "highlights": ["Q4 <b>budget</b> <b>meeting</b> scheduled for..."]
    }
  ],
  "total": 5
}
```

---

## Agent

### Chat with Agent

```http
POST /api/v1/agent/chat
Content-Type: application/json

{
  "message": "What's on my calendar today?",
  "session_id": "session-123"
}
```

**Response:**
```json
{
  "response": "You have 3 meetings today:\n\n1. 9:00 AM - Team standup (30 min)\n2. 11:00 AM - 1:1 with Sarah (45 min)\n3. 2:00 PM - Product review (1 hour)\n\nWould you like me to prepare anything for these meetings?",
  "session_id": "session-123",
  "actions_taken": [],
  "memories_used": [
    {
      "type": "episodic",
      "summary": "Retrieved today's calendar events"
    }
  ]
}
```

### Agent Status

```http
GET /api/v1/agent/status
```

**Response:**
```json
{
  "status": "running",
  "uptime_seconds": 86400,
  "active_watchers": 3,
  "items_processed_today": 127,
  "pending_actions": 2,
  "memory_stats": {
    "episodic_count": 1547,
    "semantic_count": 892,
    "procedural_count": 34
  }
}
```

### List Pending Actions

```http
GET /api/v1/agent/actions
```

**Response:**
```json
{
  "actions": [
    {
      "id": "action-001",
      "type": "confirm_meeting",
      "description": "Confirm attendance for Team Offsite",
      "item_id": "item-456",
      "suggested_response": "Yes, I'll attend",
      "deadline": "2025-01-16T17:00:00Z"
    }
  ]
}
```

### Execute Action

```http
POST /api/v1/agent/actions/{id}/execute
Content-Type: application/json

{
  "approved": true,
  "modification": null
}
```

### Dismiss Action

```http
POST /api/v1/agent/actions/{id}/dismiss
```

---

## Memory

### Recent Memories

```http
GET /api/v1/memory/recent
```

**Query Parameters:**
- `type` (string) - Memory type (episodic, semantic, procedural)
- `limit` (int) - Max results (default: 20)

**Response:**
```json
{
  "memories": [
    {
      "id": "mem-001",
      "type": "episodic",
      "timestamp": "2025-01-15T09:00:00Z",
      "summary": "Processed 15 emails from overnight",
      "details": {...}
    }
  ]
}
```

### Search Memories

```http
POST /api/v1/memory/search
Content-Type: application/json

{
  "query": "Sarah budget discussion",
  "types": ["episodic", "semantic"],
  "limit": 10
}
```

**Response:**
```json
{
  "results": [
    {
      "memory": {...},
      "score": 0.89,
      "source": "Oct 15 meeting notes"
    }
  ]
}
```

### Forget Memory

```http
DELETE /api/v1/memory/{id}
```

### Teach Agent

```http
POST /api/v1/memory/teach
Content-Type: application/json

{
  "type": "semantic",
  "fact": {
    "subject": "emails from school.edu",
    "predicate": "should_route_to",
    "object": "Parent hat"
  }
}
```

---

## WebSocket API

Connect to `ws://localhost:8420/api/v1/ws` for real-time updates.

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8420/api/v1/ws');
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-bearer-token'
  }));
};
```

### Subscribe to Events

```javascript
ws.send(JSON.stringify({
  type: 'subscribe',
  channels: ['items', 'agent', 'sync']
}));
```

### Event Types

#### New Item
```json
{
  "type": "item.new",
  "data": {
    "id": "item-001",
    "hat_id": "hat-001",
    "type": "email",
    "preview": "Parent-Teacher Conference invitation..."
  }
}
```

#### Item Updated
```json
{
  "type": "item.updated",
  "data": {
    "id": "item-001",
    "changes": ["hat_id", "importance"]
  }
}
```

#### Agent Action Required
```json
{
  "type": "agent.action_required",
  "data": {
    "id": "action-001",
    "type": "confirm_meeting",
    "description": "Confirm attendance for Team Offsite"
  }
}
```

#### Sync Progress
```json
{
  "type": "sync.progress",
  "data": {
    "space_id": "space-001",
    "status": "syncing",
    "progress": 0.45,
    "items_synced": 45
  }
}
```

#### Sync Complete
```json
{
  "type": "sync.complete",
  "data": {
    "space_id": "space-001",
    "items_synced": 100,
    "duration_ms": 5234
  }
}
```

#### Agent Chat Response (Streaming)
```json
{
  "type": "agent.chat.chunk",
  "data": {
    "session_id": "session-123",
    "chunk": "You have 3 meetings",
    "done": false
  }
}
```

### Send Chat via WebSocket

```javascript
ws.send(JSON.stringify({
  type: 'chat',
  session_id: 'session-123',
  message: 'What meetings do I have today?'
}));
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "ITEM_NOT_FOUND",
    "message": "Item with ID item-999 not found",
    "details": {}
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | Token valid but insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request body |
| `SPACE_CONNECTION_FAILED` | 502 | Failed to connect to external service |
| `SYNC_IN_PROGRESS` | 409 | Sync already running |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Internal server error |

---

## Rate Limits

Local API has generous limits:

| Endpoint | Limit |
|----------|-------|
| Chat | 60/minute |
| Search | 120/minute |
| Other | 600/minute |

---

## Examples

### cURL: Get today's actionable items

```bash
curl -X GET \
  'http://localhost:8420/api/v1/items?requires_action=true&since=2025-01-15T00:00:00Z' \
  -H 'Authorization: Bearer <token>'
```

### cURL: Chat with agent

```bash
curl -X POST \
  'http://localhost:8420/api/v1/agent/chat' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"message": "Summarize my unread emails", "session_id": "cli-session"}'
```

### JavaScript: WebSocket connection

```javascript
const ws = new WebSocket('ws://localhost:8420/api/v1/ws');

ws.onopen = () => {
  // Authenticate
  ws.send(JSON.stringify({ type: 'auth', token: TOKEN }));

  // Subscribe to all updates
  ws.send(JSON.stringify({
    type: 'subscribe',
    channels: ['items', 'agent', 'sync']
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  switch (msg.type) {
    case 'item.new':
      console.log('New item:', msg.data);
      break;
    case 'agent.action_required':
      showNotification(msg.data);
      break;
  }
};
```

---

**API Version:** v1
**Last Updated:** 2025-01-15
