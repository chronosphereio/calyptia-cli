#!/bin/sh
echo "getting latest operator image version"
latest=$(curl -sf https://api.github.com/repos/calyptia/core-operator-releases/releases | jq -r .[0].tag_name)

echo "getting version before latest operator image version"
beforelatest=$(curl -sf https://api.github.com/repos/calyptia/core-operator-releases/releases | jq -r .[1].tag_name)
echo "test install operator into default namespace"
./calyptia install operator 
./calyptia update operator --version "$beforelatest" --verbose
./calyptia update operator --version "$latest" --verbose
./calyptia uninstall operator 

echo "test install operator into test namespace"
./calyptia install operator --kube-namespace test
./calyptia update operator --version "$beforelatest" --kube-namespace test--verbose

kubectl create ns calyptia
kubectl -n calyptia create secret docker-registry "regcreds" \
    --docker-server="ghcr.io" \
    --docker-username="${REGISTRY_USERNAME:-calyptia-ci}" \
    --docker-password="$REGISTRY_PASSWORD" \
    --docker-email="${REGISTRY_EMAIL:-ci@calyptia.com}"

helm repo add --force-update calyptia https://calyptia.github.io/charts
helm repo update --fail-on-repo-update-fail
helm upgrade --install \
    --create-namespace --namespace "cloud" \
    --set global.imagePullSecrets[0]="regcreds" \
    --set global.pullPolicy=IfNotPresent \
    --set vivo.enabled=false \
    --set frontend.enabled=false \
    --set cloudApi.service.type="ClusterIP" \
    --set operator.enabled=false \
    --wait \
    calyptia-cloud calyptia/calyptia-standalone