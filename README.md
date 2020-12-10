# alertpusher-sensu

Prometheus alertmanager's default receiver. Implements callback to get prometheus
alerts from `alertmanager` and pushes the corresponding events to sensu via tcp socket.

On each `ping` alert, sensu client is also updated with `POST /clients` sensu api. The
following fields are pushed: ip address, device model, type and vendor, environment.


# compilation

```
git clone this repo
cd alertpusher-sensu
make
```

# usage

Available command line options:

```
./alertpusher-sensu -h
Usage: alertpusher-sensu [-dh] [-p port] [--sensu-api-port value] [--sensu-client-port value]
                         [--sensu-host value]

-d, --debug             enable debug mode
-h, --help              print (this) help message
-p, --port=port         web server listen port (default: 8086)
    --sensu-api-port=value
                        sensu api server port (to update clients) (default: 4567)
    --sensu-client-port=value
                        sensu socket server port (to send events) (default: 3030)
    --sensu-host=value  sensu socket server host (default: "localhost")
```

# alertmanager

This program can be userd with the following alertmanager config (`/etc/alertmanager.yml`):

```
global:
  resolve_timeout: 10m

route:
  receiver: "default"
  group_by: ['instance', 'alertname']
  # params explanation: https://www.robustperception.io/whats-the-difference-between-group_interval-group_wait-and-repeat_interval
  group_wait:      1m
  group_interval:  10m
  repeat_interval: 30m

receivers:
  - name: "default"
    webhook_configs:
    - url: 'http://10.64.32.30:8086/alert'
      send_resolved: true
```

The ip/port `10.64.32.30:80086` are the alertpusher-sensu's local listen address and port.
