package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var username string

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func signIn() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Masukkan username baru: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	fmt.Print("Masukkan password baru: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	user := User{Username: username, Password: password}
	userData, _ := json.Marshal(user)

	resp, err := http.Post("http://localhost:8080/register", "application/json", bytes.NewBuffer(userData))
	if err != nil {
		fmt.Println("Gagal registrasi ke server:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Akun berhasil dibuat! Silakan login.")
	} else {
		fmt.Println("Gagal registrasi:", resp.Status)
	}
}

func login() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	username, _ = reader.ReadString('\n')
	username = strings.TrimSpace(username)
	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	user := User{Username: username, Password: password}
	userData, _ := json.Marshal(user)

	resp, err := http.Post("http://localhost:8080/login", "application/json", bytes.NewBuffer(userData))
	if err != nil {
		fmt.Println("Gagal terhubung ke server untuk login:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Username setelah login:", username)
		fmt.Println("Login berhasil!")
		return true
	} else {
		fmt.Println("Login gagal:", resp.Status)
		return false
	}
}

func checkBalance() int {
	fmt.Println("Username di checkBalance:", username)

	// Set timeout untuk HTTP request
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:8080/balance?username=%s", username))
	if err != nil {
		fmt.Println("Gagal mengecek saldo:", err)
		return 0
	}
	defer resp.Body.Close()

	fmt.Println("Status code:", resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Response body:", string(body))

	if resp.StatusCode == http.StatusOK {
		var balance int
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&balance); err != nil {
			fmt.Println("Gagal membaca saldo dari server:", err)
			return 0
		}
		return balance
	} else {
		fmt.Printf("Gagal mengecek saldo. Status code: %d, Response body: %s\n", resp.StatusCode, string(body))
		return 0
	}
}

func sendMessage() {
	reader := bufio.NewReader(os.Stdin)
	reader.ReadBytes('\n')

	fmt.Print("Pesan dan Nominal (contoh: Halooooooooooo:10000000): ")
	userInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error membaca input:", err)
		return
	}
	userInput = strings.TrimSpace(userInput)

	parts := strings.Split(userInput, ":")
	if len(parts) != 2 {
		fmt.Println("Format input tidak valid. Contoh: Halooooooooooo:10000000")
		return
	}

	chatMessage := parts[0]
	inputNominal := parts[1]

	donationAmount, err := strconv.Atoi(inputNominal)
	if err != nil || donationAmount <= 0 {
		fmt.Println("Nominal harus berupa angka positif.")
		return
	}

	fmt.Printf("Anda akan mengirim pesan: %s\n", chatMessage)
	fmt.Printf("Dengan nominal donasi: Rp%d\n", donationAmount)
	fmt.Print("Apakah Anda yakin? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" {
		fmt.Println("Pesan dibatalkan.")
		return
	}

	fullMessage := fmt.Sprintf("%s:%d:%s", username, donationAmount, chatMessage)
	fmt.Println("Pesan yang akan dikirim:", fullMessage)

	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		fmt.Println("Gagal terhubung ke server untuk mengirim pesan:", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Write([]byte(fullMessage))
	if err != nil {
		fmt.Println("Gagal mengirim pesan:", err)
		return
	}

	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("Timeout menunggu response dari server")
		} else {
			fmt.Println("Gagal membaca response dari server:", err)
		}
		return
	}
	fmt.Println("Respon server:", string(response[:n]))
}

func mainMenu() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nMenu Utama")
		fmt.Println("1. Kirim Pesan")
		fmt.Println("2. Lihat Saldo")
		fmt.Println("3. Keluar")
		fmt.Print("Pilihan: ")

		optionStr, _ := reader.ReadString('\n')
		optionStr = strings.TrimSpace(optionStr)
		option, _ := strconv.Atoi(optionStr)

		switch option {
		case 1:
			sendMessage()
		case 2:
			balance := checkBalance()
			fmt.Printf("Saldo Anda saat ini: Rp%d\n", balance)
		case 3:
			fmt.Println("Keluar dari aplikasi...")
			return
		default:
			fmt.Println("Pilihan tidak valid. Masukkan angka 1, 2, atau 3.")
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n==== Aplikasi Donasi ====")
		fmt.Println("1. Sign In (Buat Akun Baru)")
		fmt.Println("2. Login")
		fmt.Println("3. Keluar")
		fmt.Print("Pilihan: ")

		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)
		choice, _ := strconv.Atoi(choiceStr)

		switch choice {
		case 1:
			signIn()
		case 2:
			if login() {
				mainMenu()
			}
		case 3:
			fmt.Println("Keluar dari aplikasi...")
			return
		default:
			fmt.Println("Pilihan tidak valid. Masukkan angka 1, 2, atau 3.")
		}
	}
}
