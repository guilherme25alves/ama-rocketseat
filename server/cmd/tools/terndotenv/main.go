package main

import (
	"os/exec"

	"github.com/joho/godotenv"
)

/*
	Por default, o tern não tem uma configuração que permite importar as variáveis de ambiente com arquivo .env

	Esse é o motivo da criação desse pacote em GO, que faz um wrap do tern e executa fazendo o load das variáveis de ambiente
	a partir da biblioteca godotenv, que carrega as variáveis a partir de um arquivo .env

	Dessa forma possibilitamos a importação via arquivo .env ao tern.

	Para executar, via terminal, utilizamos como um arquivo comum do GO

		go run cmd/tools/terndotenv/main.go
*/
func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"tern",
		"migrate",
		"--migrations",
		"./internal/store/pgstore/migrations",
		"--config",
		"./internal/store/pgstore/migrations/tern.conf",
	)

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
