#!/bin/bash
# Pull from git
git pull origin master
# Build
go build -o "tickle-stock-watcher.out"
echo "[TickleStockWatcher] Build Complete"
# Run Cloud Proxy
#./cloud_sql_proxy -instances=ticklemeta-203110:asia-east1:ticklemeta-coininfodb=tcp:3306 &
echo "[TickleStockWatcher] Cloud SQL Proxy On"
# Run
LISTS=$(pmgo list)
if [[ $LISTS == *"tickle-stock-watcher"* ]]; then
	pmgo stop tickle-stock-watcher
fi
pmgo start $(pwd -P)/tickle-stock-watcher.out tickle-stock-watcher true --args="-credential" --args="$(pwd -P)/credee.json" --args="-telegram" --args="$(pwd -P)/telegrambotconfig.json"
