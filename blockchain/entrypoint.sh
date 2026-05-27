#!/bin/sh
set -eu

DATADIR="/root/.ethereum"
CHAIN_DATA="$DATADIR/geth/chaindata"
ETHERBASE_NO_PREFIX="$(echo "$ETHERBASE_ADDRESS" | sed 's/^0x//' | tr '[:upper:]' '[:lower:]')"

mkdir -p "$DATADIR/keystore"

if ls "$DATADIR/keystore"/*"$ETHERBASE_NO_PREFIX" >/dev/null 2>&1; then
  echo "Admin keystore found"
else
  echo "Admin keystore not found, importing local demo accounts"

  for key_file in /accounts/*.key; do
    echo "Importing account from $key_file"
    geth --datadir "$DATADIR" account import --password /password.txt "$key_file"
  done
fi

if [ ! -d "$CHAIN_DATA" ]; then
  echo "Initializing private Ethereum network"

  geth --datadir "$DATADIR" init /genesis.json
fi

exec geth \
  --datadir "$DATADIR" \
  --networkid 2025 \
  --nodiscover \
  --syncmode full \
  --ipcdisable \
  --http \
  --http.addr 0.0.0.0 \
  --http.port 8545 \
  --http.api eth,net,web3,personal,miner,txpool,clique \
  --http.vhosts "*" \
  --http.corsdomain "*" \
  --allow-insecure-unlock \
  --unlock "$ETHERBASE_ADDRESS" \
  --password /password.txt \
  --mine \
  --miner.etherbase "$ETHERBASE_ADDRESS" \
  --miner.gasprice 0
