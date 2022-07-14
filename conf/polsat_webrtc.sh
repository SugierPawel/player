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
video_buffer_size=1M
audio_buffer_size=120K
video_pkt_size=1200
audio_pkt_size=1200

video_out="rtp://@"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$video_localrtcpport"&buffer_size="$video_buffer_size"&pkt_size="$video_pkt_size
audio_out="rtp://@"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$audio_localrtcpport"&buffer_size="$audio_buffer_size"&pkt_size="$audio_pkt_size

echo ""
echo "video_in="$video_in
echo "audio_in="$audio_in
echo "video_out="$video_out
echo "audio_out="$audio_out
echo ""

main_threads=1
thread_queue_size=512

v4l2loopback-ctl set-fps 25 /dev/video"$device_in_nr"

ffmpeg \
-hide_banner \
-threads $main_threads \
-thread_queue_size $thread_queue_size \
-fflags +genpts+nobuffer+igndts \
-i "$video_in" \
-f alsa \
-thread_queue_size $thread_queue_size \
-fflags +genpts+nobuffer+igndts \
-i "$audio_in" \
-map 0:v:0 \
-c:v copy \
-f rtp -payload_type 96 "$video_out" \
-map 1:a:0 \
-c:a libopus -compression_level 10 -frame_duration 10 -apply_phase_inv 0 -strict -2 -ac 2 -b:a 16k \
-f rtp -payload_type 97 "$audio_out"
