#!/bin/bash
#./polsat_webrtc.sh ./polsat_webrtc_1 0 224.11.11.1 1111

ffmpeg=$1
device_in_nr=$2
address_out=$3
port=$4

video_in="/dev/video"$device_in_nr
audio_in="hw:2,0,"$device_in_nr
rtcpport=$(($port + 1))
video_localrtcpport=$(($port + 2))
audio_localrtcpport=$(($port + 3))

ttl=1
buffer_size=1M
fifo_size=8192

video_out="rtp://"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$video_localrtcpport"&buffer_size=188000&pkt_size=1200"
audio_out="rtp://"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$audio_localrtcpport"&buffer_size=188000&pkt_size=1200"

echo ""
echo "video_in="$video_in
echo "audio_in="$audio_in
echo "video_out="$video_out
echo "audio_out="$audio_out
echo ""

main_threads=1
thread_queue_size=2048

v4l2loopback-ctl set-fps 25 /dev/video"$device_in_nr"

ffmpeg \
-hide_banner \
-threads $main_threads \
-thread_queue_size $thread_queue_size \
-re \
-i "$video_in" \
-f alsa \
-thread_queue_size $thread_queue_size \
-i "$audio_in" \
-map 0:v:0 \
-c:v copy \
-f rtp "$video_out" \
-map 1:a:0 \
-c:a opus -strict -2 -ac 1 \
-f rtp "$audio_out"