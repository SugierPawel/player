#!/bin/bash

ffmpeg=$1
address_in=$2
address_out=$3
video_port=$4
audio_port=$5

ttl=10
buffer_size=4194304

in="udp://@"$address_in"?pkt_size=1316&buffer_size=$buffer_size" \
video_out="rtp://"$address_out":"$video_port"?ttl="$ttl"&buffer_size="$buffer_size"&pkt_size=1200&fifo_size="$buffer_size"&overrun_nonfatal=1" \
audio_out="rtp://"$address_out":"$audio_port"?ttl="$ttl"&pkt_size=1200"

echo ""
echo "in="$in
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

$ffmpeg \
-strict -2 \
-hide_banner \
-fflags +genpts \
-fflags +discardcorrupt \
-threads $main_threads \
-max_delay 5000000 \
-thread_queue_size $thread_queue_size \
-hwaccel_device 0 \
-hwaccel cuda \
-hwaccel_output_format cuda \
-resize "$out_w"x"$out_h" \
-dn \
-sn \
-copytb 0 -start_at_zero \
-f mpegts \
-c:v h264_cuvid \
-rtbufsize 300M \
-r 25 -i "$in" \
-map 0:a:0 -map -0:s -map -0:v -map -0:d \
-c:a opus -strict -2 -b:a 48k -max_delay 0 \
-f rtp "$audio_out" \
-map 0 -map -0:s -map -0:a -map -0:d \
-metadata service_name=$ffmpeg \
-max_interleave_delta 0 -flush_packets 0 \
-c:v h264_nvenc -preset 12 -tune 1 -profile:v main -forced-idr 1 -coder cabac -g 25 -b:v "$bitrate" -cbr 1 -multipass 2 -2pass 1 -rc cbr -bufsize:v "$bufsize" \
-minrate "$minrate" -maxrate "$maxrate" -muxrate  "$muxrate" \
-f rtp "$video_out"