
### Prereq

1. install prysmctl
2. set up cluster
3. auth to cluster

### Create Genesis

prysmctl testnet generate-genesis --fork=bellatrix --num-validators=4 --output-ssz=./consensus/genesis.ssz --chain-config-file=./consensus/config.yml --geth-genesis-json-in=./execution/genesis.json --geth-genesis-json-out=./execution/genesis.json


Start execution client (runs until merge):
helm install geth geth

Start becon client/validator
helm install prysm prysm


### Docs

https://docs.prylabs.network/docs/advanced/proof-of-stake-devnet

