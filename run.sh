#!/bin/bash
# Pull from git
git pull origin dev
# Build
go build
# Run Cloud Proxy
#./cloud_sql_proxy -instances=ticklemeta-203110:asia-east1:ticklemeta-coininfodb=tcp:3306 &
# Run
pmgo start ./ tickle-stock-watcher # --args="-credential" --args="./credee.json"