#!/bin/bash

service polsat_webrtc_1 stop
service polsat_webrtc_2 stop
service polsat_webrtc_3 stop

service polsat_player stop
service coturn stop


service polsat_webrtc_1 start
service polsat_webrtc_2 start
service polsat_webrtc_3 start

service coturn start
service polsat_player start