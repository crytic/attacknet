#!/bin/bash




export DEPOSIT_CONTRACT=0x4242424242424242424242424242424242424242
export FROM=0x123463a4b065722e99115d6c222f267d9cabb524
export SELECTOR="deposit(bytes,bytes,bytes,bytes32)"
export KEYSTORE=./geth/execution/keys.json
export CHAIN_ID=3151908
export VALIDATOR_START=30000
export VALIDATOR_END=60000
export PRIVKEY="2e0834786285daccd064ca17f1654f67b4aef298acbb82cef9ec422fb4975622"
export MANIFEST_FILE=deposit_manifest.json

export KUBE_EXECUTION_SERVICE=el-1-geth-lighthouse
export EXEC_POD_PORT=8545

rm -f $MANIFEST_FILE
eth2-val-tools deposit-data \
  --source-min=$VALIDATOR_START \
  --source-max=$VALIDATOR_END \
  --amount=32000000000 \
  --fork-version="0x10000038" \
  --withdrawals-mnemonic=$PRIVKEY \
  --validators-mnemonic=$PRIVKEY \
  --as-json-list > $MANIFEST_FILE

kubectl  port-forward svc/$KUBE_EXECUTION_SERVICE $EXEC_POD_PORT &
port_forward_pid=$!
sleep 1

nonce=$(cast nonce --rpc-url http://localhost:$EXEC_POD_PORT $FROM)
count=0

while IFS= read -r line; do
    ((valIdx=VALIDATOR_START+count))
    echo "processing validator $valIdx"
    ((count=count+1))

    deposit_data_root=$(jq -r '.deposit_data_root' <<< "$line")
    signature=$(jq -r '.signature' <<< "$line")
    withdrawal_credentials=$(jq -r '.withdrawal_credentials' <<< "$line")
    pubkey=$(jq -r '.pubkey' <<< "$line")


    while true; do
      echo "sending to rpc"
      cast send \
          --keystore $KEYSTORE --password "" \
          --from $FROM \
          --nonce $nonce \
          --chain $CHAIN_ID \
          --rpc-url http://localhost:$EXEC_POD_PORT \
          --value $(cast --to-wei 32) \
            $DEPOSIT_CONTRACT $SELECTOR "$pubkey" "$withdrawal_credentials" "$signature" "$deposit_data_root" --async &

        if [ $? -eq 0 ]; then
        echo "deposit success"
            break
        else
            sleep 1
        fi
      done
      ((nonce=nonce+1))
# cast call --chain $CHAIN_ID --from $FROM --rpc-url http://localhost:$EXEC_POD_PORT 0x4242424242424242424242424242424242424242 "get_deposit_root()"
done < <(jq -c '.[]' "$MANIFEST_FILE")

kill $port_forward_pid

