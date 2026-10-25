#!/bin/sh
# Certbot deploy hook: grant the service access to renewed TLS files, then reload
# them by restarting PixelBonus. Run once during installation as well.
set -eu

lineage=/etc/letsencrypt/live/pixelbonus.com
if [ "${RENEWED_LINEAGE:-$lineage}" != "$lineage" ]; then
    exit 0
fi

setfacl -m u:pixelbonus:--x \
    /etc/letsencrypt \
    /etc/letsencrypt/live \
    /etc/letsencrypt/archive \
    "$lineage" \
    /etc/letsencrypt/archive/pixelbonus.com
setfacl -m u:pixelbonus:r-- \
    "$(readlink -f "$lineage/fullchain.pem")" \
    "$(readlink -f "$lineage/privkey.pem")"

systemctl try-restart pixelbonus.service
