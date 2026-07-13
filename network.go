package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// WakeOnLan sends a magic packet to the given MAC address
func WakeOnLan(macAddress string) error {
	macAddress = strings.ReplaceAll(macAddress, ":", "")
	macAddress = strings.ReplaceAll(macAddress, "-", "")

	if len(macAddress) != 12 {
		return fmt.Errorf("invalid MAC address length")
	}

	mac, err := hex.DecodeString(macAddress)
	if err != nil {
		return err
	}

	// Magic packet is 6 bytes of 0xFF followed by 16 repetitions of the MAC address
	var packet bytes.Buffer
	packet.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	for i := 0; i < 16; i++ {
		packet.Write(mac)
	}

	// Send UDP packet to local broadcast address (br-lan)
	addr, err := net.ResolveUDPAddr("udp", "192.168.1.255:9")
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet.Bytes())
	return err
}

// Ping checks if a given IP and Port are reachable
func Ping(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// StartInternetMonitor periodically checks internet connectivity
func StartInternetMonitor(telegramBot *TelegramBot) {
	log.Println("Starting Internet Monitor...")
	
	client := http.Client{Timeout: 5 * time.Second}
	wasDown := false
	downSince := time.Time{}

	for {
		time.Sleep(30 * time.Second)

		// Simple HEAD request to a highly available server
		resp, err := client.Head("https://1.1.1.1")
		isDown := err != nil || resp.StatusCode >= 500

		if resp != nil {
			resp.Body.Close()
		}

		if isDown {
			if !wasDown {
				wasDown = true
				downSince = time.Now()
				log.Println("Internet connection lost.")
			}
		} else {
			if wasDown {
				wasDown = false
				duration := time.Since(downSince).Round(time.Second)
				log.Printf("Internet connection restored after %s", duration)
				if telegramBot != nil {
					telegramBot.SendMessage(fmt.Sprintf("🟢 Conexión a internet restaurada. Tiempo de caída: %s", duration))
				}
			}
		}
	}
}
