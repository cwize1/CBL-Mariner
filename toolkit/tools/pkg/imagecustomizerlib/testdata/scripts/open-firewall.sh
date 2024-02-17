set -e

iptables -P INPUT ACCEPT
iptables -P OUTPUT ACCEPT
iptables-save -f /etc/systemd/scripts/ip4save
