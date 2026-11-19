package controller

// TriggerNodeImage is the fixed wait-only utility image used by trigger nodes
// in graph runs and scenario runs. Update it when publishing or replacing the
// trigger image used by the operator.
const TriggerNodeImage = "quay.io/prubenda/kraken-hub:trigger"
