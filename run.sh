#!/bin/bash
# Pull from git
git pull origin master
# Build
go build
echo "[TickleStockWatcher] Build Complete"
# Run Cloud Proxy
#./cloud_sql_proxy -instances=ticklemeta-203110:asia-east1:ticklemeta-coininfodb=tcp:3306 &
echo "[TickleStockWatcher] Cloud SQL Proxy On"
# Run
LISTS=$(pmgo list)
if [[ $LISTS == *"tickle-stock-watcher"* ]]; then
	pmgo delete tickle-stock-watcher
fi
pmgo start ./ tickle-stock-watcher --args="-credential" --args="$(pwd)/credee.json" --args="-telegram" --args="$(pwd)/telegrambotconfig.json"
