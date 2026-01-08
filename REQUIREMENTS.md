# Overview
This is an operator that will manage the orchestration of the chaos scenarios exposing
a REST API to interact with a frontend where the chaos will be orchestrated. The Operator will
also comunicate via gRPC with a python container which service will be located in the
krkn-operator-data-provider folder, this container will be deployed together with the operator 
container in the same pod.

# API Requirements

- The operator must expose a REST API

## GET methods
### /clusters
- this method will return the list of available target clusters
- this method will take a CR id as parameter
- the available clusters will be fetched from the cluster CR KrknTarget request
  that will be named after the id passed as a parameter.
  if no CR with is id is present a 404 status will be returned.
  you can find the definition in the api/v1alpha1 folder.


#### refactoring
- select only KrknTargetRequests *only* if are completed
- replace the gin gonic framework with plain net/http for API serving


### /nodes
- this method will return the nodes of a specified cluster
- this method will take two parameters, CR id and cluster-name
- this method will get a secret from the configured namespace (the same logic as /clusters)
- from the secret content structure will retrieve the kubeconfig contained inside the secret structure under the `cluster-name` object
- for the moment the method will return the kubeconfig (this will be refactored)

### /targets/{UUID} ✅ COMPLETED
- this method will return 404 if the UUID is not found
- this method will return 100 if the UUID is found but the status is not Completed
- this method will return 200 if the UUID is found and Completed

## POST methods

### /targets ✅ COMPLETED

- this method will create a new KrknTargetRequest CR in the same operator namespace with:
  - metadata.name = generated UUID
  - spec.uuid = generated UUID
- this method will return a 102 status and the UUID of the KrknTargetRequest


# Grpc python service requirement

## get nodes
- in the krkn-operator-data-provider I want to create a grpc python service that must interact with the go api
- the service must have `krkn-lib` pypi module along with the others (for the moment reference the git branch https://github.com/krkn-chaos/krkn-lib.git@init_from_string when a new version will be released we'll switch to the pypi version) 
- the get nodes will receive as a grpc parameter the kubeconfig in base64 format
- the get nodes will initialize the KrknKubernetes python object with the `kubeconfig_string` named parameter containing the decoded kubeconfig
- the get nodes will call the list_nodes() method on the KrknKubernetes object and that will return a list of strings
- the list of strings will be returned to the Go API
- the list of strings will be returned to the client
- remove the kubeconfig return parameter from the go api