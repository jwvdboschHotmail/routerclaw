import pexpect

child = pexpect.spawn("ssh -oHostKeyAlgorithms=+ssh-rsa -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null root@192.168.1.1")
i = child.expect(['password:', pexpect.EOF, pexpect.TIMEOUT], timeout=10)
if i == 0:
    child.sendline(password)
    child.expect('#', timeout=10)
    
    print("=== DMESG ===")
    child.sendline("dmesg | tail -n 20")
    child.expect('#', timeout=10)
    print(child.before.decode('utf-8'))
    
    print("=== LOGREAD ===")
    child.sendline("logread | tail -n 20")
    child.expect('#', timeout=10)
    print(child.before.decode('utf-8'))
else:
    print("Error SSH.")
