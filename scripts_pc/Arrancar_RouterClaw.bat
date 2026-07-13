@echo off
setlocal
set SSH_PASS=%ROUTERCLAW_SSH_PASSWORD%
if "%SSH_PASS%"=="" (
    echo ERROR: Define ROUTERCLAW_SSH_PASSWORD antes de ejecutar.
    echo Usa: set ROUTERCLAW_SSH_PASSWORD=tu_password
    pause
    exit /b 1
)
echo ========================================================
echo   RECONECTAR ROUTERCLAW TRAS APAGON DEL ROUTER
echo ========================================================
echo.
echo Transfiriendo cerebro de RouterClaw a la memoria RAM del router...
wsl -e bash -c "sshpass -p '%SSH_PASS%' ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'killall routerclaw' || true; sshpass -p '%SSH_PASS%' scp -O -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no /home/usuario/routerclaw/routerclaw root@192.168.1.2:/tmp/ && sshpass -p '%SSH_PASS%' ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no root@192.168.1.2 'chmod +x /tmp/routerclaw && cd /tmp && ./routerclaw /root/config.json </dev/null >/tmp/routerclaw.log 2>&1 &'"
echo.
echo ¡Listo! RouterClaw vuelve a estar vivo y corriendo en segundo plano.
pause
