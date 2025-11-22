package main

import (
	"fmt"
	"log"
	"os"
	"prueba/server"

	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, usando variables de entorno del sistema")
	}

	port := os.Getenv("portback")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	srv := server.New(addr)

	log.Printf("Servidor iniciado en http://localhost%s", addr)
	err := srv.ListenAndServe()

	if err != nil {
		panic(err)
	}
}
