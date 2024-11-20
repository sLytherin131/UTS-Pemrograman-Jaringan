package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func topUp() {
	// Membuat koneksi UDP
	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:8082")
	if err != nil {
		fmt.Println("Gagal resolve UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Gagal terhubung ke UDP server:", err)
		return
	}
	defer conn.Close()

	// Meminta input username dan jumlah top-up dari pengguna
	reader := bufio.NewReader(os.Stdin) // Gunakan bufio.NewReader
	var username string
	var amount int

	fmt.Print("Username: ")
	username, _ = reader.ReadString('\n') // Baca seluruh baris input
	username = strings.TrimSpace(username)
	fmt.Print("Jumlah Top Up: ")
	fmt.Scan(&amount)

	// Mengirim data top-up ke server
	message := fmt.Sprintf("%s:%d", username, amount)
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Gagal mengirim data:", err)
		return
	}

	// Menerima respon dari server
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Gagal menerima respon:", err)
		return
	}

	// Menampilkan respon dari server
	response := string(buffer[:n])
	fmt.Println(response)
}

func main() {
	for {
		fmt.Println("\n==== Top Up Saldo ====")
		topUp()
		fmt.Println("Tekan Ctrl+C untuk keluar atau tunggu untuk top-up lagi.")
	}
}
