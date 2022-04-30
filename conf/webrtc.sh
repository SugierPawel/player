#!/bin/bash

ffmpeg=$1
address_in=$2
address_out=$3
port=$4

rtcpport=$(($port))
video_localrtcpport=$(($port + 1))
audio_localrtcpport=$(($port + 2))

ttl=1
buffer_size=4194304

in="udp://@"$address_in"?pkt_size=1316&buffer_size=$buffer_size" \
video_out="rtp://"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$video_localrtcpport"&pkt_size=1200" \
audio_out="rtp://"$address_out":"$port"?ttl="$ttl"&rtcpport="$rtcpport"&localrtcpport="$audio_localrtcpport"&pkt_size=133"

echo ""
echo "in="$in
echo "video_out="$video_out
echo "audio_out="$audio_out
echo ""

main_threads=1
thread_queue_size=4096

out_w=640
out_h=360

bitrate=500k
minrate=500k
maxrate=500k
muxrate=500k
bufsize=500k

#-fflags +genpts \
#-fflags +nofillin \
#-fflags +discardcorrupt \
#-copytb 0 -start_at_zero \

$ffmpeg \
-strict -2 \
-hide_banner \
-threads $main_threads \
-thread_queue_size $thread_queue_size \
-hwaccel_device 0 \
-hwaccel cuda \
-hwaccel_output_format cuda \
-resize "$out_w"x"$out_h" \
-dn \
-sn \
-f mpegts \
-c:v h264_cuvid \
-rtbufsize 400M \
-i "$in" \
-map 0 -map -0:s -map -0:a -map -0:d \
-metadata service_name=$ffmpeg \
-c:v h264_nvenc -preset 12 -tune 1 -profile:v baseline -forced-idr 1 -coder cabac -b:v "$bitrate" -cbr 1 -multipass 2 -2pass 1 -rc cbr -bufsize:v "$bufsize" \
-minrate "$minrate" -maxrate "$maxrate" -muxrate  "$muxrate" \
-f rtp "$video_out" \
-map 0:a:0 -map -0:s -map -0:v -map -0:d \
-c:a opus -strict -2 -b:a 48k \
-f rtp "$audio_out"
