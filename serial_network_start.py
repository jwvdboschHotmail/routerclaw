import serial
import time
import sys

def find_and_start_network():
    ports = [f"/dev/ttyS{i}" for i in range(1, 15)]
    for port in ports:
        try:
            ser = serial.Serial(port, 115200, timeout=2)
            print(f"Probando {port}...")
            
            # Send Enter a few times to wake up the console
            ser.write(b'\r\n\r\n')
            time.sleep(1)
            
            output = ser.read(1024).decode('utf-8', errors='ignore')
            if "root@" in output or "OpenWrt" in output or "#" in output:
                print(f"¡Consola encontrada en {port}!")
                
                print("Iniciando servidor DHCP y red...")
                ser.write(b'/etc/init.d/dnsmasq start\r\n')
                time.sleep(2)
                
                ser.write(b'/etc/init.d/network restart\r\n')
                time.sleep(2)
                
                ser.close()
                return True
            ser.close()
        except Exception as e:
            pass
            
    return False

if __name__ == "__main__":
    if find_and_start_network():
        print("Éxito. Esperando 10 segundos para que Windows obtenga IP...")
        time.sleep(10)
        sys.exit(0)
    else:
        print("No se encontró la consola Serial del router.")
        sys.exit(1)
