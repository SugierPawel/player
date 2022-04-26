# nginx

        location = /signal {
                allow all;
                proxy_buffering off;

                proxy_http_version 1.1;

                keepalive_disable none;
                keepalive_timeout 60s;

                proxy_intercept_errors on;

                add_header Access-Control-Allow-Origin *;
                add_header Access-Control-Allow-Methods POST,GET,OPTIONS;
                add_header Access-Control-Allow-Headers *;

                proxy_set_header Connection "upgrade";
                proxy_set_header Upgrade "Websocket";

                proxy_set_header X-Forwarded-Host $host:$server_port;
                proxy_set_header X-Forwarded-Server $host;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

                proxy_pass_request_headers on;
                proxy_pass_request_body on;

                proxy_read_timeout 60s;
                proxy_send_timeout 60s;
                proxy_connect_timeout 60s;

                proxy_pass http://172.26.9.100:2000/signal;
        }
        location ~* /.*.*$ {
                allow all;
                proxy_http_version 1.1;
                chunked_transfer_encoding off;
                proxy_buffering off;
                proxy_cache     off;

                keepalive_disable none;
                keepalive_timeout 0s;

                proxy_read_timeout     3s;
                proxy_connect_timeout  3s;

                root /home/go/src/github.com/SugierPawel/player/www/;
        }