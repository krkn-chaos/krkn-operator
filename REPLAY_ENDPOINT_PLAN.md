# Piano Implementazione: GET /scenarios/run/replay/{jobId}

**Issue**: tsebastiani-1vs7  
**Data**: 2026-07-29  
**Tipo**: Feature Enhancement

## Obiettivo

Implementare un nuovo endpoint REST che permetta di "rifare" (replay) l'esecuzione di uno scenario completato partendo dal suo `jobId`. L'endpoint deve ricostruire il payload originale dello scenario ed essere pronto per rilanciarlo tramite `POST /scenarios/run`.

---

## Analisi Requisiti

### 1. Endpoint Specificato
```
GET /scenarios/run/replay/{jobId}
```

### 2. Flusso Logico

```
Richiesta → Verifica RBAC → Trova Pod → Estrai ScenarioRun → Ricostruisci Payload → Risposta
```

**Step dettagliati**:

1. **Input**: `jobId` (path parameter)
2. **Recupero Pod**: Filtrare i pod nel namespace con label `krkn-job-id={jobID}`
3. **Validazione**: Verificare che esista esattamente 1 pod con quel label
4. **Estrazione ScenarioRun**: Dal campo `metadata.ownerReferences` del pod, estrarre il riferimento `KrknScenarioRun/{name}`
5. **Lettura CRD**: Recuperare il CR `KrknScenarioRun` con il nome trovato
6. **Verifica RBAC**:
   - **Admin**: nessuna limitazione
   - **User**: deve appartenere ad un gruppo con permesso `run` sul cluster target (ottenibile solo DOPO aver letto lo ScenarioRun)
7. **Ricostruzione Payload**: Mappare i campi del `KrknScenarioRun.Spec` al formato `ScenarioRunRequest` usato da `POST /scenarios/run`
8. **Gestione Volumes**: **PROBLEMA CRITICO** - i volumes montati nel pod non sono persistiti nel CRD

---

## ✅ BUONE NOTIZIE: Volume Mounts Sono Persistiti!

### Analisi CRD Reale dal Cluster

Ho verificato un `KrknScenarioRun` reale sul cluster `acm-hub-krkn`:

**CRD Spec Include**:
```yaml
spec:
  registryName: test-private-registry
  registryURL: quay.io
  scenarioRepository: rh_ee_tsebasti/krkn-hub-private
  username: rh_ee_tsebasti+robot
  # password/token NOT stored (security - è in un Secret)
  
  scenarioImage: node-cpu-hog  # Short form
  files: []  # Se presenti, sono qui
  environment:
    END: "10"
    EXIT_STATUS: "0"
```

**Status Include**:
```yaml
status:
  clusterJobs:
    - containerImage: quay.io/rh_ee_tsebasti/krkn-hub-private:node-cpu-hog  # Full path!
```

**Pod Volumes** (dal pod reale `krkn-job-c080b0f9-b511-406d-b48d-21cf9de6a287`):
1. **kubeconfig** - ConfigMap auto-generato dal controller
2. **tmp** - EmptyDir (standard, non serve persistere)
3. **kube-api-access** - Service account projection (auto K8s)
4. **imagePullSecrets** - Secret per registry privato (auto-generato dal controller)

**✅ CONCLUSIONE**: Il CRD `Files []FileMount` ESISTE e contiene tutti i file custom. I volumes standard (kubeconfig, tmp, service account) vengono ricreati automaticamente dal controller.

### Verifica Necessaria

Prima di decidere se modificare il CRD, dobbiamo verificare:

1. ✅ **Il controller popola correttamente `Files` quando crea il pod?**
   - Controllare `internal/controller/krknscenariorun_controller.go`
   - Verificare se i volumes del pod vengono mappati in `spec.files`

2. ✅ **Il campo `Files` contiene TUTTI i mount necessari?**
   - Files custom dell'utente
   - Kubeconfig
   - Files di configurazione scenario
   - Files per resiliency score (se abilitato)

3. ❓ **Esistono altri tipi di volume non coperti?**
   - EmptyDir
   - ConfigMaps esterni
   - Secrets
   - HostPath

### Due Possibili Scenari

#### Scenario A: Files è Completo
Se il controller già popola tutto in `Files`, allora:
- ✅ **Nessuna modifica al CRD necessaria**
- ✅ Possiamo ricostruire il payload direttamente da `KrknScenarioRun.Spec.Files`
- ✅ Implementazione diretta

#### Scenario B: Files è Incompleto
Se mancano alcuni mount (es. volumes dinamici, configmaps, secrets):
- ⚠️ **Dobbiamo estendere il CRD**
- ⚠️ Aggiungere nuovi campi in `KrknScenarioRunSpec` (es. `Volumes`, `ConfigMaps`, `Secrets`)
- ⚠️ Modificare il controller per popolare questi campi
- ⚠️ Generare nuovi CRD manifests (`make manifests`)
- ⚠️ Aggiornare la documentazione

---

## Task Breakdown

### TASK 1: Investigazione Volume Mounts (PRIORITÀ ALTA)
**Obiettivo**: Determinare se serve modificare il CRD

**Sotto-task**:
1. Leggere `internal/controller/krknscenariorun_controller.go`
2. Identificare dove viene creato il pod (`pod.Spec.Volumes`, `pod.Spec.Containers[0].VolumeMounts`)
3. Verificare se TUTTI i volume mounts vengono salvati in `KrknScenarioRun.Spec.Files`
4. Identificare eventuali mount mancanti (kubeconfig, resiliency, custom configmaps)
5. **Decision Point**: Serve estendere il CRD? (Sì/No)

**Output**:
- Documento di analisi: `VOLUME_ANALYSIS.md`
- Decisione: `EXTEND_CRD = true/false`

---

### TASK 2a: Estensione CRD (SE NECESSARIO)
**Condizione**: Solo se `EXTEND_CRD = true`

**Sotto-task**:
1. Modificare `api/v1alpha1/krknscenariorun_types.go`:
   ```go
   type KrknScenarioRunSpec struct {
       // ... campi esistenti
       
       // NEW: Additional volume mounts not covered by Files
       Volumes []VolumeMount `json:"volumes,omitempty"`
   }
   
   type VolumeMount struct {
       Name       string            `json:"name"`
       Type       string            `json:"type"` // ConfigMap, Secret, EmptyDir
       MountPath  string            `json:"mountPath"`
       Items      map[string]string `json:"items,omitempty"` // For ConfigMap/Secret keys
   }
   ```

2. Aggiornare controller per popolare `Volumes` quando crea il pod
3. Rigenerare CRD manifests: `make manifests`
4. Scrivere unit test per i nuovi campi
5. Aggiornare documentazione API

**Stima**: 4-6 ore

---

### TASK 2b: Nessuna Modifica CRD (SE NON NECESSARIO)
**Condizione**: Solo se `EXTEND_CRD = false`

**Output**: Nessuna azione richiesta, procedere con TASK 3

---

### TASK 3: Implementare Business Logic - Retrieval Job e ScenarioRun

**File**: `internal/api/scenario_replay_handlers.go` (nuovo)

**Sotto-task**:
1. Creare handler `GetScenarioReplay(w http.ResponseWriter, r *http.Request)`
2. Estrarre `jobId` dal path parameter
3. Implementare `findPodByJobID(ctx, jobId) (*corev1.Pod, error)`:
   ```go
   // List pods with label selector
   podList := &corev1.PodList{}
   listOpts := []client.ListOption{
       client.InNamespace(h.namespace),
       client.MatchingLabels{"krkn-job-id": jobId},
   }
   if err := h.client.List(ctx, podList, listOpts...); err != nil {
       return nil, err
   }
   
   if len(podList.Items) == 0 {
       return nil, fmt.Errorf("no pod found with krkn-job-id=%s", jobId)
   }
   if len(podList.Items) > 1 {
       return nil, fmt.Errorf("multiple pods found with krkn-job-id=%s", jobId)
   }
   
   return &podList.Items[0], nil
   ```

4. Implementare `extractScenarioRunNameFromPod(pod *corev1.Pod) (string, error)`:
   ```go
   // Parse pod.ObjectMeta.OwnerReferences
   for _, ownerRef := range pod.OwnerReferences {
       if ownerRef.Kind == "KrknScenarioRun" {
           return ownerRef.Name, nil
       }
   }
   return "", fmt.Errorf("no KrknScenarioRun owner reference found")
   ```

5. Implementare `getScenarioRun(ctx, name) (*krknv1alpha1.KrknScenarioRun, error)`:
   ```go
   scenarioRun := &krknv1alpha1.KrknScenarioRun{}
   err := h.client.Get(ctx, types.NamespacedName{
       Name:      name,
       Namespace: h.namespace,
   }, scenarioRun)
   return scenarioRun, err
   ```

**Stima**: 2-3 ore

---

### TASK 4: Implementare RBAC Validation

**File**: `internal/api/scenario_replay_handlers.go`

**Logica**:
```go
claims := auth.GetClaimsFromContext(ctx)

// Admin bypass
if auth.IsAdmin(ctx) {
    // Skip permission checks
    goto reconstructPayload
}

// Regular user: extract cluster info from ScenarioRun
targetClusters := scenarioRun.Spec.TargetClusters
targetRequestID := scenarioRun.Spec.TargetRequestID

// Fetch KrknTargetRequest to get cluster API URLs
targetRequest := &krknv1alpha1.KrknTargetRequest{}
if err := h.client.Get(ctx, types.NamespacedName{
    Name:      targetRequestID,
    Namespace: h.namespace,
}, targetRequest); err != nil {
    return err
}

// Validate user has 'run' permission on all target clusters
if err := groupauth.ValidateScenarioRunAccess(
    ctx,
    h.client,
    claims.UserID,
    h.namespace,
    targetClusters,
    targetRequest,
); err != nil {
    writeJSONError(w, http.StatusForbidden, ErrorResponse{
        Error:   "forbidden",
        Message: err.Error(),
    })
    return
}
```

**Stima**: 1-2 ore

---

### TASK 5: Ricostruzione Payload per POST /scenarios/run

**File**: `internal/api/scenario_replay_handlers.go`

**⚠️ REQUISITO CRITICO**: Il payload DEVE essere 100% identico a quello della wizard!

**Obiettivo**: Mappare `KrknScenarioRun.Spec` → `ScenarioRunRequest` (formato identico a wizard)

**Checklist di Completezza** (basata su `ScenarioRunRequest` in types.go):
- ✅ `targetRequestId` - OBBLIGATORIO
- ✅ `targetClusters` - OBBLIGATORIO (map[string][]string)
- ✅ `scenarioName` - OBBLIGATORIO
- ✅ `scenarioImage` - OBBLIGATORIO (full image path: registry/repo:tag)
- ⚠️ `files` - **LEGACY** inline file mount (base64 content embedded)
- ⚠️ `fileReferences` - **NUOVO** file references by UUID (centrally-managed files)
- ✅ `environment` - Environment variables (SCENARIO_TYPE, SCENARIO_FILE, etc.)
- ✅ `kubeconfigPath` - Default: /home/krkn/.kube/config
- ⚠️ `registryURL`, `scenarioRepository`, `token`, `username`, `password` - Credentials registry (se non usa registryName)
- ⚠️ `registryName` - Nome registry salvato (alternativa a credentials esplicite)
- ✅ `maxRetries`, `retryBackoff`, `retryDelay` - Retry configuration

**⚠️ ATTENZIONE**: Due modalità di gestione file:
1. **LEGACY**: `files []FileMount` - Content base64 embedded direttamente nel payload
2. **NUOVO**: `fileReferences []FileReference` - UUID che referenziano ConfigMaps centralizzati

**PROBLEMA**: Quale modalità viene usata dalla wizard?
- Se la wizard usa `fileReferences`, dobbiamo ricostruire gli UUIDs (non solo il content)
- Se la wizard usa `files`, dobbiamo decodificare i files dai ConfigMaps e riencodarli in base64

**Implementazione**:
```go
func reconstructScenarioRunPayload(scenarioRun *krknv1alpha1.KrknScenarioRun) (*ScenarioRunRequest, error) {
    // Create payload matching EXACTLY the wizard output
    payload := &ScenarioRunRequest{
        // MANDATORY fields
        TargetRequestID: scenarioRun.Spec.TargetRequestID,
        TargetClusters:  scenarioRun.Spec.TargetClusters,
        ScenarioName:    scenarioRun.Spec.ScenarioName,
        ScenarioImage:   scenarioRun.Spec.ScenarioImage,
        
        // OPTIONAL but commonly set
        Environment:    scenarioRun.Spec.Environment,
        KubeconfigPath: scenarioRun.Spec.KubeconfigPath,
        
        // Retry configuration (with defaults if not set)
        MaxRetries:   scenarioRun.Spec.MaxRetries,
        RetryBackoff: scenarioRun.Spec.RetryBackoff,
        RetryDelay:   scenarioRun.Spec.RetryDelay,
    }
    
    // Files - Map ALL files from CRD
    // IMPORTANT: This MUST include all files that were in the original wizard payload:
    // - Custom scenario files uploaded by user
    // - Generated scenario config files
    // - Any other mounted files
    if len(scenarioRun.Spec.Files) > 0 {
        payload.Files = make([]FileMount, len(scenarioRun.Spec.Files))
        for i, f := range scenarioRun.Spec.Files {
            payload.Files[i] = FileMount{
                Name:      f.Name,
                Content:   f.Content,  // Already base64 encoded in CRD
                MountPath: f.MountPath,
            }
        }
    }
    
    // Registry configuration - Two mutually exclusive options:
    // Option 1: Named registry (references saved KrknOperatorPrivateRegistry)
    if scenarioRun.Spec.RegistryName != "" {
        payload.RegistryName = &scenarioRun.Spec.RegistryName
        // DO NOT include explicit credentials when using named registry
    } else {
        // Option 2: Explicit registry credentials
        // Only include non-empty values
        if scenarioRun.Spec.RegistryURL != "" {
            payload.RegistryURL = &scenarioRun.Spec.RegistryURL
        }
        if scenarioRun.Spec.ScenarioRepository != "" {
            payload.ScenarioRepository = &scenarioRun.Spec.ScenarioRepository
        }
        if scenarioRun.Spec.Token != "" {
            payload.Token = &scenarioRun.Spec.Token
        }
        if scenarioRun.Spec.Username != "" {
            payload.Username = &scenarioRun.Spec.Username
        }
        if scenarioRun.Spec.Password != "" {
            payload.Password = &scenarioRun.Spec.Password
        }
    }
    
    // IF EXTEND_CRD=true: Map additional Volumes (should NOT be needed if Files is complete)
    // if len(scenarioRun.Spec.Volumes) > 0 {
    //     return nil, fmt.Errorf("additional volumes not yet supported in replay payload")
    // }
    
    return payload, nil
}
```

**Validazione Output**:
```go
// After reconstruction, validate the payload is complete
func validateReplayPayload(payload *ScenarioRunRequest) error {
    if payload.TargetRequestID == "" {
        return fmt.Errorf("missing targetRequestId in reconstructed payload")
    }
    if len(payload.TargetClusters) == 0 {
        return fmt.Errorf("missing targetClusters in reconstructed payload")
    }
    if payload.ScenarioName == "" {
        return fmt.Errorf("missing scenarioName in reconstructed payload")
    }
    if payload.ScenarioImage == "" {
        return fmt.Errorf("missing scenarioImage in reconstructed payload")
    }
    
    // Validate registry configuration is complete
    if payload.RegistryName == nil || *payload.RegistryName == "" {
        // If not using named registry, at least RegistryURL should be set
        if payload.RegistryURL == nil || *payload.RegistryURL == "" {
            return fmt.Errorf("missing registry configuration (neither registryName nor registryURL set)")
        }
    }
    
    return nil
}
```

**Stima**: 3 ore (include validazione completezza payload)

---

### TASK 6: Response Type e API Documentation

**File**: `internal/api/types.go`

**⚠️ REQUISITO CRITICO**: La response DEVE essere 100% identica al payload della wizard!

**OPZIONE A - Response Minimale (CONSIGLIATA)**:
Il servizio restituisce SOLO il payload pronto per POST /scenarios/run:
```go
// GetScenarioReplay returns exactly the same type as POST /scenarios/run expects
// Response body IS the ScenarioRunRequest directly - no wrapper
```

**Response**: Direttamente `ScenarioRunRequest` (già definito in types.go)

**Esempio Response**:
```json
{
  "targetRequestId": "target-abc123",
  "targetClusters": {
    "krkn-operator": ["cluster1", "cluster2"]
  },
  "scenarioName": "pod-scenarios",
  "scenarioImage": "quay.io/krkn-chaos/krkn-hub:pod-scenarios-latest",
  "files": [
    {
      "name": "scenario.yaml",
      "content": "base64encodedcontent...",
      "mountPath": "/home/krkn/scenarios/scenario.yaml"
    }
  ],
  "environment": {
    "SCENARIO_TYPE": "pod_scenarios",
    "SCENARIO_FILE": "/home/krkn/scenarios/scenario.yaml"
  },
  "kubeconfigPath": "/home/krkn/.kube/config",
  "maxRetries": 3,
  "retryBackoff": "exponential",
  "retryDelay": "10s"
}
```

**Frontend Usage**:
```typescript
// Frontend può fare direttamente:
const replayPayload = await fetch(`/api/v1/scenarios/run/replay/${jobId}`).then(r => r.json());

// E poi usarlo immediatamente:
await fetch('/api/v1/scenarios/run', {
  method: 'POST',
  body: JSON.stringify(replayPayload)
});
```

**OPZIONE B - Response con Metadata (SE NECESSARIO)**:
Solo se il frontend ha bisogno di metadata aggiuntivo (es. per UI):
```go
// ScenarioReplayResponse wraps the payload with optional metadata
type ScenarioReplayResponse struct {
    // Payload is the EXACT request ready for POST /scenarios/run
    Payload ScenarioRunRequest `json:"payload"`
    
    // Metadata (optional) - NOT sent to POST /scenarios/run
    Metadata *ScenarioReplayMetadata `json:"_metadata,omitempty"`
}

type ScenarioReplayMetadata struct {
    JobID                  string       `json:"jobId"`
    ScenarioRunName        string       `json:"scenarioRunName"`
    OriginalOwner          string       `json:"originalOwner,omitempty"`
    OriginalStartTime      *metav1.Time `json:"originalStartTime,omitempty"`
    OriginalCompletionTime *metav1.Time `json:"originalCompletionTime,omitempty"`
    PodName                string       `json:"podName,omitempty"`
}
```

**⚠️ IMPORTANTE**: Prefisso `_metadata` con underscore per indicare che NON fa parte del payload POST.

**Scelta Consigliata**: **OPZIONE A** - Response minimale, frontend estrae metadata dal jobId se necessario.

**Swagger Annotations**:
```go
// GetScenarioReplay handles GET /api/v1/scenarios/run/replay/{jobId}
//
// @Summary Replay a scenario from a completed job
// @Description Retrieve the scenario configuration from a completed job and return a payload ready for re-execution
// @Tags scenarios
// @Produce json
// @Param jobId path string true "Job ID (krkn-job-id label value)"
// @Success 200 {object} ScenarioReplayResponse "Scenario replay configuration"
// @Failure 400 {object} ErrorResponse "Invalid job ID or validation error"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Job or ScenarioRun not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run/replay/{jobId} [get]
```

**Stima**: 1 ora

---

### TASK 7: Routing e Integrazione

**File**: `internal/api/server.go`

**Modifiche**:
```go
// In setupRoutes() o equivalente
mux.HandleFunc("/api/v1/scenarios/run/replay/", h.authMiddleware(h.GetScenarioReplay))
```

**Nota**: Il trailing `/` è importante per catturare il path parameter

**Stima**: 30 minuti

---

### TASK 8: Unit Tests

**File**: `internal/api/scenario_replay_handlers_test.go` (nuovo)

**Test Cases**:
1. ✅ **Success Case - Admin User**
   - Admin richiede replay di job valido
   - Verifica payload ricostruito correttamente
   - Status: 200 OK

2. ✅ **Success Case - Regular User con Permessi**
   - User con permesso `run` su cluster target
   - Verifica RBAC validation passa
   - Status: 200 OK

3. ❌ **Failure - Job Non Trovato**
   - jobId inesistente
   - Nessun pod con label `krkn-job-id={jobId}`
   - Status: 404 Not Found

4. ❌ **Failure - ScenarioRun Non Trovato**
   - Pod esiste ma ScenarioRun CR è stato cancellato
   - Status: 404 Not Found

5. ❌ **Failure - RBAC Denied**
   - User senza permesso `run` su cluster target
   - Status: 403 Forbidden

6. ❌ **Failure - Multiple Pods con Stesso JobID**
   - Anomalia: 2+ pods con stesso label
   - Status: 500 Internal Server Error

7. ✅ **Edge Case - Pod Senza OwnerReference**
   - Pod non ha `ownerReferences` di tipo `KrknScenarioRun`
   - Status: 400 Bad Request

8. ✅ **Edge Case - Ricostruzione Payload Completo**
   - ScenarioRun con Files, Environment, Registry custom
   - Verifica tutti i campi mappati correttamente

**Stima**: 4 ore

---

### TASK 9: Integration Test

**File**: `internal/api/scenario_replay_integration_test.go` (nuovo)

**Scenario End-to-End**:
1. Creare un `KrknTargetRequest` completo
2. Creare un `KrknScenarioRun` che referenzia il target request
3. Simulare creazione di un pod con label `krkn-job-id=test-job-123` e `ownerReferences` che punta allo ScenarioRun
4. Chiamare `GET /scenarios/run/replay/test-job-123`
5. Verificare response contiene payload completo e corretto
6. (Opzionale) Usare il payload per chiamare `POST /scenarios/run` e verificare che crei un nuovo ScenarioRun

**Stima**: 2-3 ore

---

### TASK 10: Documentazione

**File**: `docs/api/scenario-replay.md` (nuovo)

**Contenuti**:
1. Descrizione funzionalità
2. Endpoint specification
3. Request/Response examples
4. RBAC requirements
5. Error handling
6. Use cases

**Aggiornare**:
- `README.md` - Aggiungere riferimento al nuovo endpoint
- `docs/swagger.json` - Auto-generato da annotations

**Stima**: 1-2 ore

---

## Dependency Graph

```
TASK 1 (Investigation)
  |
  ├─> TASK 2a (Extend CRD) [IF NEEDED]
  |     └─> TASK 3 (Business Logic)
  |
  └─> TASK 2b (No CRD Change) [IF NOT NEEDED]
        └─> TASK 3 (Business Logic)

TASK 3 (Business Logic)
  └─> TASK 4 (RBAC Validation)
        └─> TASK 5 (Payload Reconstruction)
              └─> TASK 6 (Response Types)
                    ├─> TASK 7 (Routing)
                    ├─> TASK 8 (Unit Tests)
                    ├─> TASK 9 (Integration Tests)
                    └─> TASK 10 (Documentation)
```

---

## Stima Totale

### Scenario Migliore (No CRD Extension)
- TASK 1: 2 ore
- TASK 2b: 0 ore
- TASK 3: 3 ore
- TASK 4: 2 ore
- TASK 5: 2 ore
- TASK 6: 1 ora
- TASK 7: 0.5 ore
- TASK 8: 4 ore
- TASK 9: 3 ore
- TASK 10: 2 ore

**Totale**: ~19-20 ore (2.5 giorni lavorativi)

### Scenario Peggiore (CRD Extension Required)
- TASK 1: 2 ore
- TASK 2a: 6 ore
- TASK 3: 3 ore
- TASK 4: 2 ore
- TASK 5: 3 ore (più complesso con volumes)
- TASK 6: 1 ora
- TASK 7: 0.5 ore
- TASK 8: 5 ore (test più complessi)
- TASK 9: 3 ore
- TASK 10: 2 ore

**Totale**: ~27-28 ore (3.5 giorni lavorativi)

---

## Rischi e Mitigazioni

### Rischio 1: Volume Mounts Incompleti
**Probabilità**: Media  
**Impatto**: Alto  
**Mitigazione**: TASK 1 investigativo con decision point chiaro

### Rischio 2: RBAC Complexity
**Probabilità**: Bassa  
**Impatto**: Medio  
**Mitigazione**: Riutilizzare logica esistente da `groupauth.ValidateScenarioRunAccess`

### Rischio 3: Backward Compatibility CRD
**Probabilità**: Media (solo se TASK 2a necessario)  
**Impatto**: Alto  
**Mitigazione**: 
- Rendere nuovi campi opzionali (`omitempty`)
- Test di migrazione da CRD vecchio a nuovo
- Documentare versioning

### Rischio 4: Edge Cases nei Pod Labels
**Probabilità**: Bassa  
**Impatto**: Medio  
**Mitigazione**: 
- Validazione strict (esattamente 1 pod con label)
- Error handling esplicito per 0 o multiple matches

---

## Acceptance Criteria

- [ ] Endpoint `GET /scenarios/run/replay/{jobId}` funzionante e testato
- [ ] RBAC validation corretta (admin bypass, user con group permissions)
- [ ] Payload ricostruito contiene TUTTI i campi necessari per replay
- [ ] Volume mounts gestiti correttamente (o in `Files` esistente o con nuovi campi CRD)
- [ ] Unit test coverage ≥ 80%
- [ ] Integration test end-to-end passa
- [ ] Swagger documentation aggiornata
- [ ] Nessuna breaking change per API esistenti
- [ ] CRD manifests aggiornati (se TASK 2a eseguito)
- [ ] README e documentazione aggiornati

---

## Next Steps

1. ✅ **Creare beads issues per ogni TASK** (con dipendenze)
2. ✅ **Iniziare con TASK 1** (Investigation) per decision point
3. ⏸️ **Bloccare TASK 3-10 su TASK 1** (waiting for investigation result)
4. 🔄 **Review plan con team** prima di procedere con implementazione
