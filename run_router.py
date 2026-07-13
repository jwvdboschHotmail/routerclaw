import pexpect
import sys
from ssh_helper import get_ssh_password

password = get_ssh_password()

print("Conectando al router por SSH...")
child = pexpect.spawn("ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null root@192.168.1.1")

i = child.expect(['password:', pexpect.EOF, pexpect.TIMEOUT], timeout=10)
if i == 0:
    child.sendline(password)
    child.expect('#', timeout=10)
    
    print("Configurando DNS manual (8.8.8.8)...")
    child.sendline("echo 'nameserver 8.8.8.8' > /etc/resolv.conf")
    child.expect('#', timeout=5)
    
    print("Dando permisos de ejecucion...")
    child.sendline("chmod +x /tmp/routerclaw")
    child.expect('#', timeout=5)
    
    print("Ejecutando RouterClaw en segundo plano...")
    child.sendline("cd /tmp && ./routerclaw > /tmp/routerclaw.log 2>&1 &")
    child.expect('#', timeout=5)
    
    print("El programa se ha lanzado de forma independiente. Cerrando SSH.")
else:
    print("Error o timeout en SSH.")
    sys.exit(1)
