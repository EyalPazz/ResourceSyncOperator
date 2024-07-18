# Resource Sync Operator

## Description

This operator came to solve one simple solution:

Some background knowledge -

1.  We have a large amount of certificates, which are created by cert-manager
2.  We use openshift (therefore, use routes)
3.  Routes do not support a certificate secret refrence
4.  We use Argo, and therefore can't use the helm lookup function (it's a gitops anti-pattern and not supported)

I wanted something that would sync my routes with my output certificate secrets!

## Getting Started

### Prerequisites

- Access to a Kubernetes v1.11.3+ cluster.
- OLM Installed on it

### To Deploy on the cluster

Add the following catalog source:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: eyal-operators
  namespace: openshift-marketplace
spec:
  displayName: Eyal Operators
  image: docker.io/eyalhilma/resourcesyncoperator-catalog:v0.0.29
  sourceType: grpc
```

## Contributing

This is a very experimental project that i created for me to learn new concepts, but feel free to open
pull requests
