import pexpect
import sys
from ssh_helper import get_ssh_password

password = get_ssh_password()

print("Iniciando transferencia SCP...")
child = pexpect.spawn("scp -O -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null routerclaw config.json root@192.168.1.1:/tmp/")

i = child.expect(['password:', pexpect.EOF, pexpect.TIMEOUT], timeout=30)
if i == 0:
    child.sendline(password)
    child.expect(pexpect.EOF, timeout=120)
    print("Transferencia completada.")
    print(child.before.decode('utf-8'))
else:
    print("Error o timeout en SCP.")
    sys.exit(1)
