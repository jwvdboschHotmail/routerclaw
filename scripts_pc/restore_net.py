import serial
import time
import sys

def run_serial():
    try:
        ser = serial.Serial('COM8', 115200, timeout=1)
    except Exception as e:
        print(f"Error opening COM8: {e}")
        return

    # Wake up the shell
    ser.write(b'\n\n')
    time.sleep(1)
    
    commands = [
        b"uci set network.lan.ipaddr='192.168.1.2'\n",
        b"uci commit network\n",
        b"/etc/init.d/network restart\n",
        b"ifconfig br-lan 192.168.1.2 netmask 255.255.255.0 up\n",
        b"/etc/init.d/dnsmasq start\n"
    ]
    
    for cmd in commands:
        ser.write(cmd)
        time.sleep(2)
        print(ser.read_all().decode('utf-8', errors='ignore'))
        
    ser.close()
    print("Network restoration commands sent.")

if __name__ == "__main__":
    run_serial()
