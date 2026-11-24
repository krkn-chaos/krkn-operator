# Secret scenario inputs

Scenario and global input metadata mark a parameter with `secret: true`. The API
uses metadata from the selected scenario image as the source of truth. A client
cannot mark an arbitrary parameter secret by submitting `secretFields`.

Workflow and run read endpoints omit these values from executable environment
maps. Workflow reads also omit them from Studio form values, including reads
through the generic `/files` endpoint. Scenario detail endpoints omit defaults
for secret fields. Read responses include `secretFields` with a boolean per
field: `true` means a value is configured, `false` means it is not. Workflow and
graph run responses group these booleans by node ID. A missing `secretFields`
entry means that no saved state is available for that field.

To edit an existing workflow, submit the full executable graph to
`PUT /api/v1/workflows/{id}`. The server compares each node with its previous
version. Changing only secret values replaces or removes those values and keeps
the other secret inputs of that node. Omit a secret key to retain it; submit an
empty value to remove it. Changing any cleartext executable input or another
executable node setting discards all *previously stored* secret values for that
node. New secret values supplied in the same request are accepted. Other nodes
are unaffected. Changes to workflow metadata or Studio canvas positions do not
invalidate secrets.

For a graph run, `sourceWorkflowId` or `sourceGraphRunName` identifies the
workflow or run whose secret values may be reused. For a single scenario run,
`sourceScenarioRunName` identifies the source run. The server checks access to
the source before applying the same comparison and invalidation rule. A
consumer that omits a source ID must provide any needed secret values itself.
The server rejects a new save or replay if it cannot inspect current input
metadata. If the classification of a saved secret field has changed, it does
not reuse the old value.

The API stores the names of secret fields in an internal annotation on newly
saved workflows and runs. This preserves redaction if image metadata later
changes or becomes unavailable. Values remain in the workflow ConfigMap or
run resource for execution inside the trusted cluster namespace; application
read APIs never return them. Anyone with direct Kubernetes read access to
those resources is inside the trusted namespace boundary.
