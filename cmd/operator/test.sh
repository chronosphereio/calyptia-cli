#!/bin/sh
set -eu
export NAMESPACE="${NAMESPACE:-default}"
export PIPELINE_NAME="${PIPELINE_NAME:-test-pipeline}"
export MAX_ITERATIONS="${MAX_ITERATIONS:-10}"
export RETRY_TIMEOUT="${RETRY_TIMEOUT:-2s}"


echo "Creating a pipeline" 
if kubectl create -f - <<EOF
apiVersion: core.calyptia.com/v1
kind: Pipeline
metadata:
  name: $PIPELINE_NAME
  namespace: $NAMESPACE
spec:
  fluentbit:
    config: |
      pipeline:
         inputs:
            - Name: dummy
              Tag: dummy.log
         outputs:
            - Name: http
              Match: '*'
              Host: out-http
              Port: 8888
              Format: json
  kind: deployment
  ports:
  - backendPort: 8888
    frontendPort: 8888
    port: 8888
    protocol: tcp
  replicasCount: 1
  resources:
    limits: {}
    requests:
      storage:
        volume-size: 256Mi

EOF
then 
    echo "succeeded to create a pipeline"
else 
    echo "failed to create a pipline"
    exit 1
fi 

echo "Waiting for pipeline deployment to come up"
until kubectl rollout status -n "$NAMESPACE" deployment/"$PIPELINE_NAME" 
do
 sleep "$RETRY_TIMEOUT"
done 

echo "Deleting pipeline deployment to come up"
if kubectl delete deployment -n "$NAMESPACE" "$PIPELINE_NAME"
then 
    echo "delete deployment succeeded"
else 
    echo "delete deployment failed"
    exit 1
fi

echo "Checking if pipline deployment has been recreated"

counter=0


while :
do
    counter=$((counter +1))
    if [ "$counter" -ge "$MAX_ITERATIONS" ]
    then
     echo "couldn't get deployment after $counter retries"
     exit 1
    fi
    if kubectl get deploy -n "$NAMESPACE" "$PIPELINE_NAME"
    then
     echo "deployment successfully recreated"
     break
    else 
     sleep "$RETRY_TIMEOUT"
    fi
done

echo "delete config map"
if kubectl delete configmap -n "$NAMESPACE" "$PIPELINE_NAME"
then
    echo "delete configmap succeeded"
else
    echo "delete configmap failed"
    exit
fi

counter=0

while :
do
    counter=$((counter +1))
    echo "Retry nr $counter"
    if [ "$counter" -ge "$MAX_ITERATIONS" ]
    then
     echo "couldn't get configmap after $counter retries"
     exit 1
    fi
    if kubectl get configmap -n "$NAMESPACE" "$PIPELINE_NAME"
    then
     echo "configmap successfully recreated"
     break
    else 
     sleep "$RETRY_TIMEOUT"
    fi
done

counter=0
while :
do
    counter=$((counter +1))
    echo "Retry nr $counter"
    if [ "$counter" -ge "$MAX_ITERATIONS" ]
    then
     echo "couldn't get service after $counter retries"
     exit 1
    fi
    if kubectl get service -n "$NAMESPACE" "$PIPELINE_NAME"-tcp8888
    then
     echo "service successfully recreated"
     break
    else 
     sleep "$RETRY_TIMEOUT"
    fi
done

if kubectl delete pipeline -n "$NAMESPACE" "$PIPELINE_NAME" 
then 
    echo "removed the pipeline"
else
    echo "failed removing the pipeline"
    exit 1
fi 