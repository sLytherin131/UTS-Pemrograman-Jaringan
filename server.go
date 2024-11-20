package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	users     = make(map[string]string) // username:password
	balances  = make(map[string]int)    // username:balance
	messages  = make([]string, 0)
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients   = make(map[*websocket.Conn]bool) // menyimpan semua koneksi websocket
	mutex     sync.Mutex
	broadcast = make(chan string)
)

// Menangani koneksi WebSocket
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	// Kirim pesan lama ke klien baru
	for _, msg := range messages {
		conn.WriteJSON(msg)
	}

	// Kirim pesan broadcast ke klien
	for {
		msg := <-broadcast
		fmt.Println("Mengirim pesan ke WebSocket:", msg) // Debugging: Tampilkan pesan yang akan dikirim

		// Iterasi melalui setiap client
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				fmt.Println("WebSocket write error:", err) // Debugging: Tampilkan error WebSocket
				mutex.Lock()
				delete(clients, client)
				mutex.Unlock()
				client.Close()
			}
		}
	}
}

// Fungsi untuk menangani koneksi TCP dan mem-broadcast pesan ke WebSocket
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("TCP read error:", err)
		return
	}

	msg := strings.TrimSpace(string(buffer[:n]))
	fmt.Println("Pesan dari TCP:", msg) // Debugging: Tampilkan pesan yang diterima dari TCP
	data := strings.SplitN(msg, ":", 3)
	if len(data) != 3 {
		conn.Write([]byte("Pesan tidak valid: Format harus 'username:amount:message'\n"))
		return
	}

	username := strings.TrimSpace(data[0])
	amountStr := strings.TrimSpace(data[1])
	message := strings.TrimSpace(data[2])

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		conn.Write([]byte("Nilai donasi tidak valid: harus berupa angka\n"))
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	fmt.Println("Memproses pesan di handleTCPConnection") // Debugging: Tampilkan proses di handleTCPConnection

	if balance, exists := balances[username]; exists && balance >= amount {
		balances[username] -= amount
		fullMessage := fmt.Sprintf("%s: Donasi Rp%d - %s", username, amount, message)
		messages = append(messages, fullMessage)
		broadcast <- fullMessage // Kirim ke channel broadcast
		conn.Write([]byte("Donasi berhasil diproses\n"))
	} else if !exists {
		conn.Write([]byte("Pengguna tidak ditemukan\n"))
	} else {
		conn.Write([]byte("Saldo tidak mencukupi\n"))
	}
}

// Fungsi untuk menangani top-up saldo melalui UDP
func handleUDPConnection(conn *net.UDPConn) {
	buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("UDP read error:", err)
			continue
		}

		message := string(buffer[:n])
		data := strings.Split(message, ":")
		if len(data) != 2 {
			conn.WriteToUDP([]byte("Format tidak valid"), addr)
			continue
		}

		username := data[0]
		amount, err := strconv.Atoi(data[1])
		if err != nil {
			conn.WriteToUDP([]byte("Jumlah tidak valid"), addr)
			continue
		}

		mutex.Lock()
		if _, exists := balances[username]; exists {
			balances[username] += amount
			response := fmt.Sprintf("Saldo %s sekarang: Rp%d", username, balances[username])
			conn.WriteToUDP([]byte(response), addr)
		} else {
			conn.WriteToUDP([]byte("Username tidak ditemukan"), addr)
		}
		mutex.Unlock()
	}
}

// Handler untuk register pengguna baru
func registerHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()
	if _, exists := users[user.Username]; exists {
		http.Error(w, "Username sudah terdaftar", http.StatusConflict)
		return
	}

	users[user.Username] = user.Password
	balances[user.Username] = 0
	fmt.Println("User terdaftar:", user.Username)
	fmt.Println("Data users:", users)
	fmt.Println("Data balances:", balances)
	w.WriteHeader(http.StatusCreated)
}

// Handler untuk login pengguna
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if storedPassword, exists := users[user.Username]; exists && storedPassword == user.Password {
		fmt.Println("User login:", user.Username)
		fmt.Println("Data users:", users)
		fmt.Println("Data balances:", balances)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Login berhasil!")
	} else {
		http.Error(w, "Username atau password salah", http.StatusUnauthorized)
	}
}

// Handler untuk mengambil saldo pengguna
func balanceHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	fmt.Println("Username yang diminta:", username)
	fmt.Println("Data balances:", balances)

	mutex.Lock()
	defer mutex.Unlock()
	if balance, exists := balances[username]; exists {
		fmt.Println("Saldo ditemukan:", balance)
		json.NewEncoder(w).Encode(balance)
	} else {
		fmt.Println("Username tidak ditemukan di balances")
		http.Error(w, "Username tidak ditemukan", http.StatusNotFound)
	}
}

func main() {
	// Endpoint HTTP untuk WebSocket
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./templates")))

	// Endpoint HTTP untuk registrasi, saldo, dan login
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/balance", balanceHandler)
	http.HandleFunc("/login", loginHandler)

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("HTTP server error:", err)
		}
	}()
	fmt.Println("HTTP server running on :8080")

	// Server TCP untuk Kirim Pesan
	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":8081")
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Gagal menjalankan TCP server:", err)
		return
	}
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				fmt.Println("TCP accept error:", err)
				continue
			}
			go handleTCPConnection(conn)
		}
	}()
	fmt.Println("TCP server running on :8081")

	// Server UDP untuk Top Up
	udpAddr, _ := net.ResolveUDPAddr("udp", ":8082")
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Gagal menjalankan UDP server:", err)
		return
	}
	defer udpConn.Close()
	go handleUDPConnection(udpConn)
	fmt.Println("UDP server running on :8082")

	select {}
}
