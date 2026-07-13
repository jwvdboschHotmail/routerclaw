@echo off
echo ========================================================
echo   RECONECTAR ROUTERCLAW TRAS APAGON DEL ROUTER
echo ========================================================
echo.
echo Transfiriendo cerebro de RouterClaw a la memoria RAM del router...
wsl -e bash -c "sshpass -p ROUTER_SSH_PASSWORD ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'killall routerclaw' || true; sshpass -p ROUTER_SSH_PASSWORD scp -O -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no /home/usuario/routerclaw/routerclaw root@192.168.1.2:/tmp/ && sshpass -p ROUTER_SSH_PASSWORD ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'chmod +x /tmp/routerclaw && cd /tmp && ./routerclaw /root/config.json </dev/null >/tmp/routerclaw.log 2>&1 &'"
echo.
echo ¡Listo! RouterClaw vuelve a estar vivo y corriendo en segundo plano.
pause
