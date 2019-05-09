#!/bin/sh
IMAGE=k8r.eu/justjanne/bahn-proxy
TAGS=$(git describe --always --tags HEAD)
DEPLOYMENT=bahn-proxy
POD=bahn-proxy
NAMESPACE=bahn-tools

kubectl -n $NAMESPACE set image deployment/$DEPLOYMENT $POD=$IMAGE:$TAGS