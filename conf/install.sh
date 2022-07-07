#!/bin/bash

cp etc/nginx/sites-available/default /etc/nginx/sites-available/default

cp etc/rsyslog.d/polsat_player.conf /etc/rsyslog.d/polsat_player.conf
cp etc/rsyslog.d/polsat_restream_1.conf /etc/rsyslog.d/polsat_restream_1.conf
cp etc/rsyslog.d/polsat_restream_2.conf /etc/rsyslog.d/polsat_restream_2.conf
cp etc/rsyslog.d/polsat_webrtc_1.conf /etc/rsyslog.d/polsat_webrtc_1.conf
cp etc/rsyslog.d/polsat_webrtc_2.conf /etc/rsyslog.d/polsat_webrtc_2.conf

service rsyslog restart

cp etc/systemd/system/polsat_player.service /etc/systemd/system/polsat_player.service
cp etc/systemd/system/polsat_restream_1.service /etc/systemd/system/polsat_restream_1.service
cp etc/systemd/system/polsat_restream_2.service /etc/systemd/system/polsat_restream_2.service
cp etc/systemd/system/polsat_webrtc_1.service /etc/systemd/system/polsat_webrtc_1.service
cp etc/systemd/system/polsat_webrtc_2.service /etc/systemd/system/polsat_webrtc_2.service

systemctl enable polsat_player.service
systemctl enable polsat_restream_1.service
systemctl enable polsat_restream_2.service
systemctl enable polsat_webrtc_1.service
systemctl enable polsat_webrtc_2.service

cp etc/turnserver.conf /etc/turnserver.conf

chmod +x polsat_restream.sh
chmod +x polsat_webrtc.sh
chmod +x restart.sh
chmod +x install.sh

mkdir /home/scripts
cp polsat_restream.sh /home/scripts/polsat_restream.sh
cp polsat_webrtc.sh /home/scripts/polsat_webrtc.sh
cp restart.sh /home/scripts/restart.sh