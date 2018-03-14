```
docker build -t nginx-proxy-cf .
```
```
docker run --name nginx-proxy-cf --rm -d -p 443:443 \
    -v ${PWD}/vhost.d:/etc/nginx/vhost.d:ro \
    -v ${PWD}/certs:/etc/nginx/certs \
    -v /var/run/docker.sock:/tmp/docker.sock:ro \
    -e CF_API_EMAIL=<cloudflare email> \
    -e CF_API_KEY=<cloudflare api key> \
    -e PROXY_IP=<proxy ip> \
    nginx-proxy-cf
```