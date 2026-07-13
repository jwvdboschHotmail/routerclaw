import pexpect
import sys
from ssh_helper import get_ssh_password

password = get_ssh_password()

child = pexpect.spawn("ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null root@192.168.1.1")
i = child.expect(['password:', pexpect.EOF, pexpect.TIMEOUT], timeout=10)
if i == 0:
    child.sendline(password)
    child.expect('#', timeout=10)
    child.sendline("killall routerclaw")
    child.expect('#', timeout=5)
    print("RouterClaw killed on router.")
else:
    print("Error SSH.")
