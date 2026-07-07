# Resiliency Score Feature - Frontend Implementation Guide

## Overview

Il backend ha implementato la feature completa per il calcolo automatico del resiliency score nelle GraphRun. Questo documento descrive le modifiche API e fornisce linee guida per l'implementazione frontend.

## Backend Status: ✅ Completato

- ✅ API per configurazione (HTTP headers)
- ✅ CRD fields per persistenza
- ✅ Controller per env vars injection
- ✅ Calcolo automatico score da pod logs
- ✅ API responses con score e baseline
- ✅ Tests completi (API + Controller)

## API Changes

### 1. Request - Create GraphRun (POST /api/v1/graphruns)

**Nuovi HTTP Headers (opzionali):**

```http
X-Resiliency-Score: true                     # Enable resiliency score calculation
X-Resiliency-Baseline: 9.0                   # Required if enabled, must be >= 0
X-Resiliency-Mount-Path: /etc/krkn/metrics.yaml  # Optional, default internal metrics
```

**Validazione:**
- Se `X-Resiliency-Score: true`, allora `X-Resiliency-Baseline` è REQUIRED
- `X-Resiliency-Baseline` deve essere un numero >= 0
- `X-Resiliency-Mount-Path` deve essere un path assoluto (opzionale)

**Errori:**
- 400 Bad Request se baseline mancante quando score enabled
- 400 Bad Request se baseline negativa
- 400 Bad Request se mount path relativo

### 2. Response - List GraphRuns (GET /api/v1/graphruns)

**Nuovi campi in `GraphRunListItem`:**

```typescript
interface GraphRunListItem {
  // ... existing fields ...
  
  // Resiliency score configuration
  resiliencyScoreEnabled?: boolean;        // Score calculation enabled
  resiliencyScoreBaseline?: number;        // User-defined baseline target
  
  // Resiliency score result (populated when run completes)
  resiliencyScore?: ResiliencyScoreResponse;
}
```

**Esempio Response:**

```json
{
  "graphRuns": [
    {
      "name": "graphrun-abc123",
      "phase": "Completed",
      "summary": {
        "totalNodes": 3,
        "completedNodes": 3
      },
      "resiliencyScoreEnabled": true,
      "resiliencyScoreBaseline": 9.0,
      "resiliencyScore": {
        "calculated": 91.5,
        "baseline": 9.0,
        "status": "pass",
        "message": "Score 91.50 meets baseline 9.00"
      }
    }
  ]
}
```

### 3. Response - Get GraphRun Detail (GET /api/v1/graphruns/:name)

**Nuovi campi in `GraphRunSpecResponse`:**

```typescript
interface GraphRunSpecResponse {
  // ... existing fields ...
  
  // Resiliency score configuration
  resiliencyScoreEnabled?: boolean;
  resiliencyMountPath?: string;           // Where metrics file is mounted
  resiliencyScoreBaseline?: number;
}
```

**Nuova struttura `ResiliencyScoreResponse` (già esistente in status):**

```typescript
interface ResiliencyScoreResponse {
  calculated: number;        // Final calculated score (0-100)
  baseline?: number;         // User-defined baseline (same as spec)
  status: string;           // "pass" | "fail" | "no-baseline"
  message?: string;         // Human-readable result
}
```

**Esempio Response:**

```json
{
  "name": "graphrun-abc123",
  "spec": {
    "graph": { ... },
    "resiliencyScoreEnabled": true,
    "resiliencyMountPath": "/etc/krkn/metrics.yaml",
    "resiliencyScoreBaseline": 9.0
  },
  "status": {
    "phase": "Completed",
    "resiliencyScore": {
      "calculated": 91.5,
      "baseline": 9.0,
      "status": "pass",
      "message": "Score 91.50 meets baseline 9.00"
    }
  }
}
```

## Frontend Implementation Guidelines

### Phase 1: Display Score in List View

**Priorità: Alta**

Mostrare lo score nella lista GraphRuns per permettere agli utenti di vedere rapidamente i risultati.

**UI Suggestions:**
```tsx
// GraphRun list item
<GraphRunCard>
  <Title>graphrun-abc123</Title>
  <Status phase="Completed" />
  
  {/* NEW: Resiliency Score Badge */}
  {graphRun.resiliencyScoreEnabled && (
    <ResiliencyScoreBadge
      score={graphRun.resiliencyScore?.calculated}
      baseline={graphRun.resiliencyScoreBaseline}
      status={graphRun.resiliencyScore?.status}
    />
  )}
</GraphRunCard>
```

**Badge States:**
- `status: "pass"` → Green badge "✓ 91.5 / 9.0"
- `status: "fail"` → Red badge "✗ 7.2 / 9.0"
- `status: "no-baseline"` → Blue badge "91.5 (no baseline)"
- Score not calculated yet → Gray badge "Calculating..."

### Phase 2: Display Score in Detail View

**Priorità: Alta**

Mostrare configurazione completa e risultato nella vista dettaglio.

**UI Suggestions:**
```tsx
// GraphRun detail page
<DetailSection title="Resiliency Score">
  {spec.resiliencyScoreEnabled ? (
    <>
      <ConfigRow label="Enabled" value="Yes" />
      <ConfigRow label="Baseline" value={spec.resiliencyScoreBaseline} />
      <ConfigRow label="Metrics File" value={spec.resiliencyMountPath || "Default"} />
      
      {status.resiliencyScore && (
        <ResultCard
          calculated={status.resiliencyScore.calculated}
          baseline={status.resiliencyScore.baseline}
          status={status.resiliencyScore.status}
          message={status.resiliencyScore.message}
        />
      )}
    </>
  ) : (
    <EmptyState message="Resiliency score not enabled for this run" />
  )}
</DetailSection>
```

### Phase 3: Configuration UI (Modal)

**Priorità: Media (vedi beads tasks)**

Implementare l'UI per configurare resiliency score quando si crea una GraphRun.

**Beads Tasks Creati:**
- `tsebastiani-lg1w` - Epic: Resiliency Score Feature - Frontend
- `tsebastiani-upgh` - FileSelector component
- `tsebastiani-4vun` - ResiliencyScoreModal component
- `tsebastiani-f6rt` - GraphRun integration
- `tsebastiani-de6d` - Tests

**Modal Flow:**
1. User clicca checkbox "Enable Resiliency Score" nel form GraphRun creation
2. Si apre modal con:
   - Input baseline (float, required, >= 0)
   - FileSelector per file metriche (opzionale)
     - Modalità: "Same file for all nodes" o "Per-node file"
   - Input mount path (default `/etc/krkn/metrics.yaml`)
3. User conferma → headers inviati in richiesta POST

**Request Example:**
```typescript
const response = await fetch('/api/v1/graphruns', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Resiliency-Score': 'true',
    'X-Resiliency-Baseline': '9.0',
    'X-Resiliency-Mount-Path': '/etc/krkn/metrics.yaml'
  },
  body: JSON.stringify({
    graph: { /* ... */ },
    targetRequestId: '...',
    targetClusters: { /* ... */ }
  })
});
```

### Phase 4: Filtering & Sorting

**Priorità: Bassa**

Permettere agli utenti di filtrare/ordinare GraphRuns per score.

**Features:**
- Filter: "Only with resiliency score enabled"
- Filter: "Passed" / "Failed" / "All"
- Sort: "Score (high to low)" / "Score (low to high)"
- Sort: "Baseline delta" (how far from baseline)

## Type Definitions (TypeScript)

```typescript
// Add to src/types/api.ts

export interface GraphRunListItem {
  name: string;
  namespace: string;
  creationTimestamp: string;
  phase: string;
  ownerUserId: string;
  targetRequestId: string;
  summary: GraphRunSummaryResponse;
  startTime?: string;
  completionTime?: string;
  
  // NEW: Resiliency score fields
  resiliencyScoreEnabled?: boolean;
  resiliencyScoreBaseline?: number;
  resiliencyScore?: ResiliencyScoreResponse;
}

export interface GraphRunSpecResponse {
  graph: Record<string, GraphScenarioNode>;
  targetRequestId: string;
  targetClusters: Record<string, string[]>;
  ownerUserId: string;
  
  // NEW: Resiliency score configuration
  resiliencyScoreEnabled?: boolean;
  resiliencyMountPath?: string;
  resiliencyScoreBaseline?: number;
}

export interface ResiliencyScoreResponse {
  calculated: number;
  baseline?: number;
  status: "pass" | "fail" | "no-baseline";
  message?: string;
}
```

## Example: Resiliency Score Badge Component

```tsx
import React from 'react';
import { Badge } from '@patternfly/react-core';
import { CheckCircleIcon, TimesCircleIcon, InfoCircleIcon } from '@patternfly/react-icons';

interface ResiliencyScoreBadgeProps {
  score?: number;
  baseline?: number;
  status?: string;
}

export const ResiliencyScoreBadge: React.FC<ResiliencyScoreBadgeProps> = ({
  score,
  baseline,
  status
}) => {
  // Score not calculated yet
  if (!score) {
    return (
      <Badge color="grey" icon={<InfoCircleIcon />}>
        Calculating...
      </Badge>
    );
  }

  // Score calculated
  const scoreText = baseline 
    ? `${score.toFixed(1)} / ${baseline.toFixed(1)}`
    : score.toFixed(1);

  switch (status) {
    case 'pass':
      return (
        <Badge color="green" icon={<CheckCircleIcon />}>
          ✓ {scoreText}
        </Badge>
      );
    case 'fail':
      return (
        <Badge color="red" icon={<TimesCircleIcon />}>
          ✗ {scoreText}
        </Badge>
      );
    default:
      return (
        <Badge color="blue" icon={<InfoCircleIcon />}>
          {scoreText} {!baseline && '(no baseline)'}
        </Badge>
      );
  }
};
```

## Testing

### Manual Testing Steps

1. **Create GraphRun with resiliency score:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/graphruns \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -H "X-Resiliency-Score: true" \
     -H "X-Resiliency-Baseline: 9.0" \
     -H "X-Resiliency-Mount-Path: /etc/krkn/metrics.yaml" \
     -d '{
       "graph": { ... },
       "targetRequestId": "...",
       "targetClusters": { ... }
     }'
   ```

2. **Verify list response includes fields:**
   ```bash
   curl http://localhost:8080/api/v1/graphruns \
     -H "Authorization: Bearer $TOKEN"
   ```

3. **Verify detail response includes fields:**
   ```bash
   curl http://localhost:8080/api/v1/graphruns/graphrun-abc123 \
     -H "Authorization: Bearer $TOKEN"
   ```

4. **Wait for GraphRun completion** and verify `resiliencyScore` populated

### Expected Behavior

- **During run:** `resiliencyScore` is `null`
- **After completion:** `resiliencyScore` contains calculated result
- **Score calculation:** Happens automatically when GraphRun reaches terminal state
- **Immutability:** Once set, `resiliencyScore` never changes (historical record)

## Backend Implementation Details

Per comprendere meglio il funzionamento backend:

1. **Headers → CRD Fields:**
   - API handler valida headers e popola `Spec.ResiliencyScoreEnabled`, `Spec.ResiliencyScoreBaseline`, `Spec.ResiliencyMountPath`

2. **Controller → Env Vars:**
   - Controller inietta `RESILIENCY_SCORE=true` in tutti i pod
   - Se `ResiliencyMountPath` specificato e file trovato → `RESILIENCY_FILE=<path>`

3. **Pod Logs → Score:**
   - Ogni pod scrive `KRKN_RESILIENCY_REPORT_JSON:{...}` nei log
   - Controller fetcha logs quando GraphRun completa
   - Usa `krknctl` package per parsing e aggregazione
   - Calcola score finale e popola `Status.ResiliencyScore`

4. **Pass/Fail Logic:**
   - `calculated >= baseline` → `status: "pass"`
   - `calculated < baseline` → `status: "fail"`
   - `no baseline` → `status: "no-baseline"`

## Questions?

Per domande o chiarimenti:
- Check beads epic: `tsebastiani-lg1w`
- Review backend code: `internal/controller/krkngraphrun_resiliency.go`
- Review API code: `internal/api/graphrun_handlers.go`

---

**Document Version:** 1.0  
**Last Updated:** 2026-07-07  
**Backend Branch:** `resiliency_score`
