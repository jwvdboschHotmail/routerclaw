import os

def get_ssh_password():
    password = os.environ.get("ROUTERCLAW_SSH_PASSWORD")
    if password:
        return password
    config_path = os.path.expanduser("~/.routerclaw/ssh_password")
    if os.path.isfile(config_path):
        with open(config_path) as f:
            return f.read().strip()
    return input("SSH password for router: ")
