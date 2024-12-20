#!/usr/bin/env bash

CHAIN_CMD="layerd"

usage() {
  echo "Usage: $0 -n <number_of_accounts> -m <mnemonics_output_file> -g <genesis_output_file> -j <existing_json_file>"
  exit 1
}

while getopts "n:m:g:j:" opt; do
  case $opt in
    n) NB_ACCOUNTS=$OPTARG ;;
    m) MNEMONICS_OUTPUT=$OPTARG ;;
    j) EXISTING_JSON=$OPTARG ;;
    *) usage ;;
  esac
done

if [[ -z "$NB_ACCOUNTS" || -z "$MNEMONICS_OUTPUT" ]]; then
  usage
fi

if ! [[ "$NB_ACCOUNTS" =~ ^[0-9]+$ ]] || [ "$NB_ACCOUNTS" -le 0 ]; then
  echo "Error: Number of accounts must be a positive integer."
  usage
fi

WORKDIR=$(mktemp -d)
MNEMONICS_FILE="${WORKDIR}/mnemonics.txt"
COUNTER=1

EXISTING_JSON="./chains/layer.json"

trap 'rm -rf -- "$WORKDIR"' EXIT

# Overwrite mnemonic output file if it exists
> "$MNEMONICS_OUTPUT"

echo "Generating $NB_ACCOUNTS mnemonics..."

ACCOUNTS_JSON=[]
ACCOUNTS=()

for ((i=1; i <= NB_ACCOUNTS; i++)); do
  MNEMONIC=$($CHAIN_CMD keys mnemonic 2>/dev/null)
  KEY="user$COUNTER"
  ADDR=$(echo "$MNEMONIC" | $CHAIN_CMD keys add $KEY --keyring-backend memory --recover --home="$WORKDIR" 2>/dev/null | grep "address" | awk '{print $3}')
  
  printf "%s\n" "$MNEMONIC" >> "$MNEMONICS_OUTPUT"
  ACCOUNTS+=("$ADDR")
  
  # Add account details to accounts JSON structure
  ACCOUNTS_JSON=$(echo "$ACCOUNTS_JSON" | jq --arg name "$KEY" --arg addr "$ADDR" --arg mnemonic "$MNEMONIC" --arg amount "10000000000%DENOM%" '.[. | length] |= . + {
    "name": $name,
    "address": $addr,
    "amount": $amount,
    "mnemonic": $mnemonic
  }')
  ((COUNTER++))
done

# Append accounts to the `accounts` array in the existing JSON file
jq --argjson accounts "$ACCOUNTS_JSON" '.chains[0].genesis.accounts = $accounts' "$EXISTING_JSON" > "$EXISTING_JSON.tmp" \
  && mv "$EXISTING_JSON.tmp" "$EXISTING_JSON"


echo "Updated JSON file: $EXISTING_JSON"
echo "Mnemonics saved to: $MNEMONICS_OUTPUT"
