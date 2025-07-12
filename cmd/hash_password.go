package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Pedir la contraseña por consola
	fmt.Print("Escribe la nueva contraseña: ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	password = password[:len(password)-1] // quitar el salto de línea

	// Generar el hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Println("Error generando hash:", err)
		os.Exit(1)
	}

	fmt.Println("Hash generado:")
	fmt.Println(string(hash))
}
