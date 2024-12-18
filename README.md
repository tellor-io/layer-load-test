## Build

```bash
go build -o ./build/load-tester ./cmd/load-tester/main.go
```

## Run

```bash
./build/load-tester -c 1 -T 10 -r 100 -s 250  \
--broadcast-tx-method async --endpoints ws://localhost:26657/websocket \
--stats-output ./stats.csv
```

## Tools

The `scripts` directory contains the `generate_accounts.sh` script, which can be used to generate accounts and mnemonics for the load test.

```bash
# Generate 25 accounts 
# The `mnenomics.txt` file will contain the mnemonics of the generated accounts, one per line. 
# The `genesis_balance.json` file will contain the genesis balance segment for each account.
# The `layer.json` file is the local devnet configuration where the accounts will be added to genesis.
./scripts/generate_accounts.sh -n 25 -m mnemonics.txt -g genesis_balance.json -j ~/layer/local_devnet/chains/layer.json