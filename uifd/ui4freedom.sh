#!/bin/sh /etc/rc.common

START=96
STOP=10
USE_PROCD=1

APP_PATH="${UIF_SERVICE_PATH:-/usr/bin/uif/uif}"

start_service() {
    [ -x "$APP_PATH" ] || return 1
    procd_open_instance
    procd_set_param command "$APP_PATH"
    procd_set_param respawn
    procd_close_instance
}

stop_service() {
    procd_kill "$NAME"
}

restart() {
    stop_service
    start_service
}

reload() {
    restart
}
