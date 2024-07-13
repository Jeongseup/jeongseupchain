#!/bin/bash

set -ux

# setup vars
NODE_DAEMON=ludiumappd
NODE_DENOM=stake

# reset previous init
rm -r ~/.$NODE_DAEMON || true
NODE_DAEMON=$(which $NODE_DAEMON)

# start message 
echo "=== this init script for cosmos-sdk v0.45.4 ==="
echo 

# configure minid
$NODE_DAEMON config chain-id demo
$NODE_DAEMON config keyring-backend test

echo "=== create alice account ==="
$NODE_DAEMON keys add alice
echo "=== create bob account ==="
$NODE_DAEMON keys add bob

echo "=== init chain ==="
$NODE_DAEMON init testmoniker --chain-id demo

# update genesis
echo "=== add genesis account for alice ==="
$NODE_DAEMON add-genesis-account alice 100000000$NODE_DENOM --keyring-backend test
# echo "=== add genesis account for bob ==="
# $NODE_DAEMON add-genesis-account bob 1000$NODE_DENOM --keyring-backend test

# create default validator
echo "=== gentx for alice validator ==="
$NODE_DAEMON gentx alice 100000000$NODE_DENOM --chain-id demo
echo "=== collect gentxs ==="
$NODE_DAEMON collect-gentxs