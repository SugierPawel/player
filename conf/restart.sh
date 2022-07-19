#!/bin/bash

service polsat_webrtc_1 stop
service polsat_webrtc_2 stop
service polsat_restream_1 stop
service polsat_restream_2 stop

service polsat_player stop
service coturn stop

rm -rf /var/log/polsat_player.log
rm -rf /var/log/polsat_restream_1.log
rm -rf /var/log/polsat_restream_2.log
rm -rf /var/log/polsat_webrtc_1.log
rm -rf /var/log/polsat_webrtc_2.log

touch /var/log/polsat_player.log
touch /var/log/polsat_restream_1.log
touch /var/log/polsat_restream_2.log
touch /var/log/polsat_webrtc_1.log
touch /var/log/polsat_webrtc_2.log

cd /usr/local/go/src/github.com/SugierPawel/player/
git pull origin master
rm -rf player
go build

service polsat_webrtc_1 start
service polsat_webrtc_2 start

service coturn start
service polsat_player start

tail -f /var/log/polsat_player.log