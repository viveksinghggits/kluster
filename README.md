

# Kluster (In Progress)

An operator that we are writing in the YouTube channel (https://youtu.be/89PdRvRUcPU)

viveksingh.dev
v1alpha1

generate

1. deep copy objects
2. clientset
3. informers
4. lister

# Commands that we used

Should be run from root dir of project.

## Code generator

Make sure you have installed code generator

```
execDir=~/go/src/k8s.io/code-generator
"${execDir}"/generate-groups.sh all github.com/viveksinghggits/kluster/pkg/client github.com/viveksinghggits/kluster/pkg/apis viveksingh.dev:v1alpha1 --go-header-file "${execDir}"/hack/boilerplate.go.txt
```

## CRDs

```
controller-gen paths=github.com/viveksinghggits/kluster/pkg/apis/viveksingh.dev/v1alpha1  crd:trivialVersions=true  crd:crdVersions=v1  output:crd:artifacts:config=manifests
```