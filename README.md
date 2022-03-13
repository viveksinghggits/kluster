
# Kluster

A example Kubernetes operator to create Kubernetes cluster on DigitalOcean.
Once the operator is running, and we create a Kluster K8S resource in a cluster, a DigitalOcean Kubernetes
cluster would be created with provided configuration.

This operator was written as part of one of my [YouTube playlist](https://www.youtube.com/watch?v=89PdRvRUcPU&list=PLh4KH3LtJvRTtFWz1WGlyDa7cKjj2Sns0).

Here is an example of the Kluster resource

```yaml
apiVersion: viveksingh.dev/v1alpha1
kind: Kluster
metadata:
  name: kluster-0
spec:
  name: kluster-0
  region: "nyc1"
  version: "1.21.3-do.0"
  tokenSecret: "default/dosecret" # secret that has DigitalOcean token
  nodePools:
    - count: 3
      name: "dummy-nodepool"
      size: "s-2vcpu-2gb"
```

# Deploy on a Kubernetes cluster

Execute below command, from root of the repo

Create Kluster CRD

```sh
kubectl create -f manifests/viveksingh.dev_klusters.yaml
```

Create RBAC resources and deployment

```sh
kubectl create -f manifests/install/
```

# Create a secret with DigitalOcean token

To call DigitalOcean APIs we will have to create a secret with DigitalOcean token that
will be used in the Kluster CR that we create.

```sh
kubectl create secret generic dosecret --from-literal token=<your-DO-token>
```

# Create a kluster CR

Create the kluster resource to create a k8s cluster in DigitalOcean

```sh
kubectl create -f manifests/klusterone.yaml
```

# Updating CRD manifest

Update the CRD manifest (`manifests/viveksingh.dev_klusters.yaml`) using docker:

```sh
docker buildx build --file hack/Dockerfile.update-crds -o . .
```

# Updating the auto-generated code

Update the auto-generated code (`pkg/client/` and `pkg/apis/viveksingh.dev/v1alpha1/zz_generated.deepcopy.go`) using docker:

```sh
docker buildx build --file hack/Dockerfile.update-codegen -o . .
```
