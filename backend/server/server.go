package server

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró archivo .env, usando variables de entorno del sistema")
	}
}

// Middleware CORS
func corsMiddleware(next http.Handler) http.Handler {
	urlfront := os.Getenv("urlfront")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Origen permitido: tu frontend en Vite
		w.Header().Set("Access-Control-Allow-Origin", urlfront)
		w.Header().Set("Vary", "Origin")

		// Métodos permitidos
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		// Headers permitidos (IMPORTANTE: Content-Type para tu POST /sync)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Peticiones preflight (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Para GET/POST normales, sigue la cadena
		next.ServeHTTP(w, r)
	})
}

func New(addr string) *http.Server {
	// Aquí registras tus rutas
	// initRoutes seguramente hace algo tipo:
	// http.HandleFunc("/item", getItem)
	// http.HandleFunc("/sync", sincItems)
	initRoutes()

	// Usas el DefaultServeMux, pero envuelto con CORS
	handlerConCORS := corsMiddleware(http.DefaultServeMux)

	return &http.Server{
		Addr:    addr,
		Handler: handlerConCORS,
	}
}
