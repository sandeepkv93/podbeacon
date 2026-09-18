#!/bin/bash
kubectl get pods -n podbeacon-system
kubectl logs -n podbeacon-system -l control-plane=controller-manager --tail=200
