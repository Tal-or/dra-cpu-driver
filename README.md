## Running driver on OpenShift

#### Enable DynamicResourceAllocation Feature on FeatureGate

Add `TechPreviewNoUpgrade` to `FeatureGate` CR:

```yaml
apiVersion: config.openshift.io/v1
kind: FeatureGate
metadata:
  name: cluster 
....

spec:
  featureSet: TechPreviewNoUpgrade 
```

#### Add the privileged pod to SCC

The driver is talking to kubelet socket, hence it requires running in
a privileged mode.

Therefore, we should add the `ServiceAccount` of the pod to the `priviliged` `SecurityContextConstraints` CR:

`oc edit scc privileged`

```yaml
kind: SecurityContextConstraints
metadata:
  name: privileged
...
users:
  - system:serviceaccount:dra-cpu-driver:dra-cpu-driver-service-account

```

#### Run the Driver using helm

```
helm upgrade -i \                                 
  --create-namespace \
  --namespace dra-cpu-driver \                       
  dra-cpu-driver \
  deployments/helm/dra-cpu-driver
```