#!/bin/bash

#ExecStart=/home/scripts/./polsat_restream.sh /home/scripts/./polsat_restream_2 224.10.11.121:2222 0

ffmpeg=$1
address_in=$2
device_out_nr=$3

buffer_size=1M
fifo_size=8192

in="rtp://@"$address_in"?pkt_size=1316&buffer_size="$buffer_size"&fifo_size="$fifo_size"&overrun_nonfatal=1"
video_out="/dev/video"$device_out_nr
audio_out="hw:2,1,"$device_out_nr

echo ""
echo "in="$in
echo "video_out="$video_out
echo "audio_out="$audio_out
echo ""

main_threads=1
thread_queue_size=2048

out_w=640
out_h=360

bitrate=500k
minrate=200k
maxrate=500k
muxrate=700k
bufsize=1400k

$ffmpeg \
-hide_banner \
-threads $main_threads \
-thread_queue_size $thread_queue_size \
-hwaccel_device 0 \
-hwaccel cuda \
-hwaccel_output_format cuda \
-rtbufsize 64M \
-f rtp \
-dn -sn \
-resize "$out_w"x"$out_h" \
-c:v h264_cuvid \
-re -i "$in" \
-metadata service_name=$ffmpeg \
-map 0:a:0 \
-c:a pcm_s16le -b:a 16k \
-f alsa "$audio_out" \
-map 0:v:0 \
-g:v 25 \
-c:v h264_nvenc -preset p5 -rc vbr -tune 2 -profile:v baseline -coder cabac -multipass 1 -2pass 1 \
-b:v "$bitrate" -bufsize:v "$bufsize" -minrate "$minrate" -maxrate "$maxrate" -muxrate "$muxrate" \
-f v4l2 "$video_out"
