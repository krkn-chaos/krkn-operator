# Privileged scenarios

Some scenarios need a privileged container. A scenario image declares this with the `krknctl.privileged` label, the label krknctl reads too, and the operator reads it when it resolves the image:

- public scenarios: from krknctl's scenario details;
- scenarios from a private registry: from the image config, over the same authenticated connection the operator uses for the image digest.

Only a scenario labeled `krknctl.privileged=true` gets a scenario container with `securityContext.privileged: true`. When the label is `false` or missing, the container is not privileged. The Pod-level settings (user, group and fsGroup 1001) are the same either way.

## Allowing privileged Pods

Scenario Pods run in the operator namespace with the `krkn-operator-krkn-scenario-runner` ServiceAccount, and the cluster must let that ServiceAccount run privileged Pods:

- **OpenShift:** the Helm chart grants the `anyuid` SCC, which does not allow privileged containers. Also grant the `privileged` SCC:

  ```bash
  oc adm policy add-scc-to-user privileged -z krkn-operator-krkn-scenario-runner -n <operator-namespace>
  ```

- **Kubernetes with Pod Security Admission:** the operator namespace must allow the `privileged` level, for example with the label `pod-security.kubernetes.io/enforce=privileged`.

If the cluster rejects a privileged Pod, the run fails with an error that names the scenario and the ServiceAccount.
