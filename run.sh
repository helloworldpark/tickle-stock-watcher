#!/bin/bash
# Pull from git
git pull origin master
# Run Cloud Proxy
#./cloud_sql_proxy -instances=ticklemeta-203110:asia-east1:ticklemeta-coininfodb=tcp:3306 &
# Run
pmgo delete tickle-stock-watcher
pmgo start $(pwd)/ tickle-stock-watcher --args="-credential" --args="$(pwd)/credee.json" --args="-telegram" --args="$(pwd)/telegrambotconfig.json"