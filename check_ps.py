import pexpect

child = pexpect.spawn("ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null root@192.168.1.1")
i = child.expect(['password:', pexpect.EOF, pexpect.TIMEOUT], timeout=10)
if i == 0:
    child.sendline(password)
    child.expect('#', timeout=10)
    
    child.sendline("ps | grep nanoclaw")
    child.expect('#', timeout=10)
    print(child.before.decode('utf-8'))
else:
    print("Error SSH.")
