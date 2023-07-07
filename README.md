
### Prereq

1. install prysmctl
2. set up cluster
3. auth to cluster


### Run goreli nodes
`helm install geth geth --values ./geth/values-goreli.yaml`
`helm install prysm prysm --values ./prysm/values-goreli.yaml`

### Private Devnet, single node

Start execution client (runs until merge):

`helm install geth geth --values ./geth/values-singlenode-64-validators.yaml --wait`

Start beacon chain client
`helm install beacon prysm --values ./prysm/values-singlenode-beacon.yaml --wait`

Start becon client/validator

`helm install validator prysm --values ./prysm/values-singlenode-validator.yaml`


### Docs

https://docs.prylabs.network/docs/advanced/proof-of-stake-devnet

### live tail k8s logs

`k logs --follow geth-0`
`k logs --follow beacon-prysm-0`
`k logs --follow validator-prysm-0`

```
helm install geth geth --values ./geth/values-singlenode-64-validators.yaml --wait && helm install beacon prysm --values ./prysm/values-singlenode-beacon.yaml --wait && helm install validator prysm --values ./prysm/values-singlenode-validator.yaml
```


```
helm uninstall geth
helm uninstall beacon
helm uninstall validator

```

## Create PVC Inspector pod
```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: pvc-inspector
spec:
  containers:
  - image: busybox
    name: pvc-inspector
    command: ["tail"]
    args: ["-f", "/dev/null"]
    volumeMounts:
    - mountPath: /pvc
      name: pvc-mount
  volumes:
  - name: pvc-mount
    persistentVolumeClaim:
      claimName: genesis-geth-0
EOF
```

`kubectl exec -it pvc-inspector -- sh`