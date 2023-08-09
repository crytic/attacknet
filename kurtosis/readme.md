kurtosis run --enclave quickstart main.star

kurtosis clean -a


kurtosis clean -a && kurtosis run --enclave quickstart .

kurtosis service shell quickstart postgres

kurtosis service logs quickstart api


kurtosis run --enclave ethTestnet github.com/kurtosis-tech/eth2-package  "$(cat ./example.json)"


interesting constraints on network topology
https://github.com/kurtosis-tech/eth-network-package/blob/main/package_io/input_parser.star



kurtosis cluster set cloud
kurtosis gateway

kurtosis engine restart --enclave-pool-size 1
kurtosis engine start --enclave-pool-size {pool-size-number}

kurtosis run --enclave ethTestnet ./eth2-package  "$(cat ./example.json)"

kubectl  port-forward svc/grafana 3000

kurtosis enclave inspect ethTestnet