alias b := build
alias r := load-test

build:
    go build -o ./build/load-tester ./cmd/load-tester/main.go
    sleep 2
    go build -o ./build/setup ./cmd/load-tester/setup/main.go

delegate:
    ./build/setup delegate

create-reporter:
    ./build/setup create-reporter

setup:
    just delegate
    sleep 5
    just create-reporter

load-test CONNECTIONS="1" TIME="10" RATE="25" SIZE="250" TXMETHOD="async" ENDPOINTS="ws://localhost:26657/websocket" OUTPUT="./stats.csv":
    ./build/load-tester -v \
    -c {{CONNECTIONS}} \
    -T {{TIME}} \
    -r {{RATE}} \
    -s {{SIZE}} \
    --broadcast-tx-method {{TXMETHOD}} \
    --endpoints {{ENDPOINTS}} \
    --stats-output {{OUTPUT}}

    #   -c The number of connections to open to each endpoint simultaneously
    #   -T The duration (in seconds) for which to handle the load test
    #   -r The number of transactions to generate each second on each connection, to each endpoint
    #   -s The size of each transaction, in bytes - must be greater than 40

load-test-manual +ARGS:
    ./build/load-tester {{ARGS}}

local-devnet:
    if command -v local-ic > /dev/null; then \
        echo "Starting local chain"; \
        ICTEST_HOME=. local-ic start layer.json; \
    else \
        echo "local-ic binary not found. Consider installing local-ic: https://github.com/strangelove-ventures/interchaintest/blob/main/local-interchain/README.md"; \
    fi

generate-accounts NUMBER_OF_ACCOUNTS:
    ./scripts/generate_accounts.sh -n {{ NUMBER_OF_ACCOUNTS }} -m mnemonics.txt