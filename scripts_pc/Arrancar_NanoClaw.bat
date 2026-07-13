@echo off
echo ========================================================
echo   RECONECTAR NANOCLAW TRAS APAGON DEL ROUTER
echo ========================================================
echo.
echo Transfiriendo cerebro de NanoClaw a la memoria RAM del router...
wsl -e bash -c "sshpass -p ROUTER_SSH_PASSWORD ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'killall nanoclaw' || true; sshpass -p ROUTER_SSH_PASSWORD scp -O -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no /home/usuario/nanoclaw/nanoclaw root@192.168.1.2:/tmp/ && sshpass -p ROUTER_SSH_PASSWORD ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'chmod +x /tmp/nanoclaw && cd /tmp && ./nanoclaw /root/config.json </dev/null >/tmp/nanoclaw.log 2>&1 &'"
echo.
echo ¡Listo! NanoClaw vuelve a estar vivo y corriendo en segundo plano.
pause
