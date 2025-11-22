package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4"
)

type Item struct {
	Ticker     string `json:"ticker"`
	TargetFrom string `json:"target_from"`
	TargetTo   string `json:"target_to"`
	Company    string `json:"company"`
	Action     string `json:"action"`
	Brokerage  string `json:"brokerage"`
	RatingFrom string `json:"rating_from"`
	RatingTo   string `json:"rating_to"`
	Time       string `json:"time"`
}

type APIResponse struct {
	Items    []Item `json:"items"`
	NextPage string `json:"next_page"`
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method not allowed")
		return
	}
	fmt.Fprintf(w, "Hello there %s", "visitor")
}

func getItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Println("Obteniendo items desde base de datos")
	dsn := os.Getenv("dsn")
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error connecting to database: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close(ctx)

	// 👇 OJO: si la columna time es TIMESTAMPTZ, la casteo a texto para que
	// encaje con el campo Time string del struct.
	rows, err := conn.Query(ctx, `
		SELECT
			ticker,
			target_from,
			target_to,
			company,
			action,
			brokerage,
			rating_from,
			rating_to,
			time::text AS time
		FROM items
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error obteniendo items: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item

	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.Ticker,
			&it.TargetFrom,
			&it.TargetTo,
			&it.Company,
			&it.Action,
			&it.Brokerage,
			&it.RatingFrom,
			&it.RatingTo,
			&it.Time,
		); err != nil {
			http.Error(w, fmt.Sprintf("Error leyendo fila: %v", err), http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error finalizando lectura: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(struct {
		Items []Item `json:"items"`
	}{
		Items: items,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error codificando respuesta: %v", err), http.StatusInternalServerError)
		return
	}
}

func obteneritemsDesdeAPI(nextPage string) ([]Item, string, error) {
	client := &http.Client{}

	url := os.Getenv("url")
	if nextPage != "" {
		url = url + "?next_page=" + nextPage
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error creating request: %w", err)
	}

	token := os.Getenv("token")
	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, "", fmt.Errorf("error parsing response JSON: %w", err)
	}

	return apiResponse.Items, apiResponse.NextPage, nil
}

func obtenerTodosLosItems() ([]Item, error) {
	var allItems []Item
	nextPage := ""

	for {
		items, np, err := obteneritemsDesdeAPI(nextPage)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if np == "" {
			break
		}
		nextPage = np
	}

	return allItems, nil
}

func insertarItemsLote(ctx context.Context, conn *pgx.Conn, items []Item) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	rows := make([][]interface{}, 0, len(items))

	for _, it := range items {
		rows = append(rows, []interface{}{
			it.Ticker,
			it.TargetFrom,
			it.TargetTo,
			it.Company,
			it.Action,
			it.Brokerage,
			it.RatingFrom,
			it.RatingTo,
			it.Time, // CockroachDB acepta RFC3339 como TIMESTAMPTZ
		})
	}

	// Insertar todo el lote con COPY
	n, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"items"},
		[]string{"ticker", "target_from", "target_to", "company", "action", "brokerage", "rating_from", "rating_to", "time"},
		pgx.CopyFromRows(rows),
	)

	return n, err
}

func sincItems(w http.ResponseWriter, r *http.Request) {
	log.Println("=== Iniciando sincronización de items ===")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method not allowed")
		return
	}

	// Paso 1: Obtener TODOS los items desde la API
	log.Println("Paso 1: Obteniendo items desde la API (todas las páginas)...")
	items, err := obtenerTodosLosItems()
	if err != nil {
		log.Printf("Error obteniendo items desde API: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error obteniendo items desde API: %v", err)
		return
	}
	log.Printf("Paso 1: Se encontraron %d items para sincronizar", len(items))

	// Paso 2: Conectar a la base de datos
	log.Println("Paso 2: Conectando a la base de datos...")
	dsn := os.Getenv("dsn")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error connecting to database: %v", err)
		return
	}
	defer conn.Close(ctx)

	// Paso 3: Crear tabla si no existe
	log.Println("Paso 3: Verificando/creando tabla items...")
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS items (
			ticker STRING,
			target_from STRING,
			target_to STRING,
			company STRING,
			action STRING,
			brokerage STRING,
			rating_from STRING,
			rating_to STRING,
			time TIMESTAMP,
			PRIMARY KEY (ticker, time)
		)
	`)
	if err != nil {
		log.Printf("Error creating table: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error creating table: %v", err)
		return
	}

	// Paso 4: Limpiar tabla (si tu intención es un full refresh)
	log.Println("Paso 4: Limpiando tabla items (TRUNCATE)...")
	_, err = conn.Exec(ctx, `TRUNCATE TABLE items`)
	if err != nil {
		log.Printf("Error truncating table: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error truncating table: %v", err)
		return
	}

	// Paso 5: Insertar items
	log.Println("Paso 5: Insertando items...")

	log.Println("Paso 5: Insertando items en lote...")

	insertedCount, err := insertarItemsLote(ctx, conn, items)

	if err != nil {
		log.Printf("Error insertando lote: %v", err)
		http.Error(w, fmt.Sprintf("Error insertando lote: %v", err), http.StatusInternalServerError)
		return
	}

	// Paso 6: Respuesta
	log.Printf("=== Sincronización completada: %d/%d items insertados ===", insertedCount, len(items))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Sincronización completada", "items_synced": %d}`, insertedCount)
}
