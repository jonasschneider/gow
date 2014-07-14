#!/bin/bash
set -eux

rm "$HOME/.pow/.path"

#
# Firewall
#
sudo launchctl unload /Library/LaunchDaemons/com.jonasschneider.gow.firewall.plist
sudo rm -f /Library/LaunchDaemons/com.jonasschneider.gow.firewall.plist

# Try to find the ipfw rule and delete it.
HTTP_PORT=20559
DST_PORT=80
RULE=$(sudo ipfw show | (grep ",$HTTP_PORT .* dst-port $DST_PORT in" || true) | cut -f 1 -d " ")
[[ -n "$RULE" ]] && sudo ipfw del "$RULE"

#
# DNS
#
sudo rm -f /etc/resolver/dev

#
# gowd
#
launchctl unload "$HOME/Library/LaunchAgents/com.jonasschneider.gow.gowd.plist"
rm -f "$HOME/Library/LaunchAgents/com.jonasschneider.gow.gowd.plist"
rm -f "$HOME/.pow/.run"


echo "*** Uninstalled"
