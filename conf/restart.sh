#!/bin/bash

service polsat_webrtc_1 stop
service polsat_webrtc_2 stop
service polsat_webrtc_3 stop

service polsat_player stop
service coturn stop

cd /home/go/src/github.com/SugierPawel/player/
git pull origin master
rm -rf player
go build

service polsat_webrtc_1 start
service polsat_webrtc_2 start
service polsat_webrtc_3 start

service coturn start
service polsat_player start

tail -f /var/log/polsat_player.log