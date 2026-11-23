export NAMESPACE=krkn-operator
export REGISTRY=quay.io/rh-ee-rashetty
export TAG=dev
export STORAGE_CLASS=""  # Update storage class provisioner


helm upgrade  krkn-operator charts/krkn-operator \
  --version 1.2.0-rc2 \
  --namespace krkn-operator-system \
  --create-namespace \
  --set console.route.enabled=true \
  --set acm.enabled=true \
  --set krknAI.enabled=true \
  --set "images.aiOrchestrator.image=$REGISTRY/krkn-ai-operator:$TAG" \
  --set "images.aiService.image=$REGISTRY/krkn-ai-service:$TAG" \
  --set aiService.storage.existingClaim=krkn-shared-file
