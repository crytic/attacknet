
### Prereq

1. install prysmctl
2. set up cluster
3. auth to cluster


### Run goreli nodes
`helm install geth geth --values ./geth/values-goreli.yaml`
`helm install prysm prysm --values ./prysm/values-goreli.yaml`

### Private Devnet, single node

Generate genesis manifest
```
prysmctl testnet generate-genesis \
    --fork=bellatrix \
    --num-validators=64 \
    --output-ssz=./prysm/consensus/genesis.ssz \
    --chain-config-file=./prysm/consensus/config.yml \
    --geth-genesis-json-in=./geth/execution/genesis.json \
    --geth-genesis-json-out=./geth/execution/genesis.json
```

Start execution client (runs until merge):

`helm install geth geth --values ./geth/values-singlenode-64-validators.yaml`

Start becon client/validator

`helm install prysm prysm --values ./prysm/values-singlenode-64-validators.yaml`


### Docs

https://docs.prylabs.network/docs/advanced/proof-of-stake-devnet

