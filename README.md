## Requirements

- [Just](https://just.systems/man/en/)
- [docker]()

## Build

```bash
just b
```

## Run

```bash
just r
```


## Flow

**when running from scratch:**

## Pre-requisite

install [local-ic](https://github.com/strangelove-ventures/interchaintest/blob/main/local-interchain/README.md)  
build the [layer](https://github.com/tellor-io/layer/blob/e875ac47d15afb13a725a615ea74fe87c8b314fe/Makefile#L238) docker image

1. create the desired number of mnemonics and accounts
```bash
just generate-accounts 25 
```
2. start the chain
```bash
just local-devnet
```
3. build the binaries
```bash
just b
```
4. populate the .env file

5. delegate accounts to validator and create reporters
```bash
just setup
```
6. run the load test
```bash
just r
```