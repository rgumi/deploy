#!/bin/sh

$NS=ingress-depoy
kubectl create ns $NS |true

kustomize edit set image depoy=erpk/depoy:$DEPOY_TAG
kustomize build | tee build.yaml

kubectl apply -f build.yaml
rm -f build.yaml
