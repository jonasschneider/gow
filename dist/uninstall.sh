#!/bin/bash
set -eu

# remove our crap
rm "$HOME/.pow/.path"
rm "$HOME/.pow/.run"

#
# Firewall
#
sudo launchctl unload /Library/LaunchDaemons/com.jonasschneider.gow.firewall.plist
sudo rm -f /Library/LaunchDaemons/com.jonasschneider.gow.firewall.plist

# Try to find the firewall rule and delete it.
sudo pfctl -a com.apple/250.PowFirewall -F nat 2>/dev/null || true

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
