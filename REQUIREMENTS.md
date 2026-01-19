# "ROAD TO PRODUCTION" REQUIREMENTS

## KrknTargetRequest controller in operator

### Overview
- Krkn Operator must be able to register itself as KrknOperatorTargetProvider via the CRD with name `krkn-operator`.
- Krkn Operator will expose an API to instantiate a new CRD called KrknOperatosTarget
- - `KrknOperatorTarget` must contain the following attributes
- - - `UUID` (String  - UIID)
- - - `ClusterName` (String)
- - - `ClusterAPIURL` (String)
- - - `SecretType` (String can assume "kubeconfig" or "token" or "credentials" values)
- - - `SecretUIID` (String - UUID)
- - - `CABundle` (String)
- krkn operator must expose a CRUD REST API for `a c` and must adhere to the following rules
- - Creation
- - - A UUID must be created for the `UUID` field 
- - - Two `KrknOperatorTarget` with the same name or the same apiUrl are not allowed
- - - if `SecretType` is token or credentials:
- - - - `ClusterAPIURL` is mandatory
- - - - `ClusterCABundle` is optional, if not set the client will skip TLS verification
- - - if `SecretType` is kubeconfig the `ClusterAPIURL` must be extracted from the kubeconfig
- - - a Secret named with an `UUID` must be created containing a valid kubeconfig created accordingly with the credentials provided
- - Retrival
- - - The kubeconfig will be *never ever passed* back to the client, all the transaction must happen with the UUID and, where needed
      the kubeconfig is resolved accordingly.

- TODO (skip for the moment): krkn operator must have a controller for KrknTargetRequest and must populate it with KrknOperatorTarget as the [krkn-operator-acm](https://github.com/krkn-chaos/krkn-operator-acm/blob/main/internal/controller/krkntargetrequest_controller.go)
  does with the same logic (please read it with a lot of attention) 
## API and operator refactoring to support multiple clusters

### Overview
TODO

## Moving from direct scenario creation to CRD approach

### Overview
TODO

