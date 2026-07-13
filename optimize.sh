#!/bin/sh
echo "Deteniendo servicios innecesarios..."
/etc/init.d/dnsmasq stop
/etc/init.d/dnsmasq disable
/etc/init.d/odhcpd stop
/etc/init.d/odhcpd disable
/etc/init.d/firewall stop
/etc/init.d/firewall disable

echo "Desactivando el bridge de LAN para ahorrar RAM..."
uci delete network.lan.type
uci commit network

echo "Memoria libre ANTES de reiniciar red:"
free

echo "Reiniciando la red en segundo plano..."
/etc/init.d/network restart >/dev/null 2>&1 &
