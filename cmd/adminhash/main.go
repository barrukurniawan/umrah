// adminhash membuat ADMIN_PASSWORD_HASH (bcrypt) untuk /admin.
// Pakai: go run ./cmd/adminhash   lalu ketik password (tidak tampil), salin hasilnya ke .env.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Fprint(os.Stderr, "Password admin: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	pw := strings.TrimRight(line, "\r\n")
	if len(pw) < 10 {
		fmt.Fprintln(os.Stderr, "\nPassword minimal 10 karakter.")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("ADMIN_PASSWORD_HASH='%s'\n", hash)
}
