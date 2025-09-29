package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Stars - структура для представления звезды
type Stars struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	AlternativeName string  `json:"alternative_name"`
	Constellation   string  `json:"constellation"`
	Distance        float32 `json:"distance"`
	Mass            float32 `json:"mass"`
}

// Response - структура для ответов API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// App - структура для хранения зависимостей приложения
type App struct {
	DB *sql.DB
}

// setupDatabase - настройка подключения к базе данных
func (app *App) setupDatabase(config *Config) error {
	var err error
	app.DB, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	// Проверяем соединение
	if err = app.DB.Ping(); err != nil {
		return fmt.Errorf("не удается подключиться к БД: %v", err)
	}

	log.Println("Успешно подключились к PostgreSQL")

	return nil
}

// healthCheck - проверка работоспособности API
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Сервер работает",
	})
}

// getAllStars - получение всех звезд
func (app *App) getAllStars(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовки для JSON ответа
	w.Header().Set("Content-Type", "application/json")

	// SQL запрос для получения всех звезд
	rows, err := app.DB.Query(
		"SELECT sn.id, sn.name, sn.alternative_name, c.name AS constellation, s.distance_light_years AS distance, s.solar_masses AS mass FROM stars s JOIN star_names sn ON s.star_name_id = sn.id JOIN constellations c on s.constellation_id = c.id")
	if err != nil {
		log.Printf("Ошибка запроса: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Ошибка получения данных",
		})
		return
	}
	defer rows.Close()

	var star_res []Stars
	// Проходим по всем строкам результата
	for rows.Next() {
		var stars Stars
		err := rows.Scan(&stars.ID, &stars.Name, &stars.AlternativeName, &stars.Constellation, &stars.Distance, &stars.Mass)
		if err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}
		star_res = append(star_res, stars)
	}

	// Отправляем успешный ответ
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Звезды получены",
		Data:    star_res,
	})
}

// getStarSearch - получение звезды по ID
func (app *App) getStarSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Структура для получения данных из POST запроса
	var searchRequest struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}

	// Декодируем JSON из тела запроса
	err := json.NewDecoder(r.Body).Decode(&searchRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Неверный формат данных",
		})
		return
	}

	// Проверяем обязательные поля
	if searchRequest.Field == "" || searchRequest.Value == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Поле field и value обязательны",
		})
		return
	}

	var query string
	switch searchRequest.Field {
	case "name":
		query = "SELECT sn.id, sn.name, sn.alternative_name, c.name AS constellation, s.distance_light_years AS distance, s.solar_masses AS mass FROM stars s JOIN star_names sn ON s.star_name_id = sn.id JOIN constellations c on s.constellation_id = c.id WHERE sn.name ILIKE $1"
	case "constellation":
		query = "SELECT sn.id, sn.name, sn.alternative_name, c.name AS constellation, s.distance_light_years AS distance, s.solar_masses AS mass FROM stars s JOIN star_names sn ON s.star_name_id = sn.id JOIN constellations c on s.constellation_id = c.id WHERE c.name ILIKE $1"
	case "distance":
		query = "SELECT sn.id, sn.name, sn.alternative_name, c.name AS constellation, s.distance_light_years AS distance, s.solar_masses AS mass FROM stars s JOIN star_names sn ON s.star_name_id = sn.id JOIN constellations c on s.constellation_id = c.id WHERE s.distance_light_years = $1"
	case "mass":
		query = "SELECT sn.id, sn.name, sn.alternative_name, c.name AS constellation, s.distance_light_years AS distance, s.solar_masses AS mass FROM stars s JOIN star_names sn ON s.star_name_id = sn.id JOIN constellations c on s.constellation_id = c.id WHERE s.solar_masses = $1"
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Неверное поле для поиска",
		})
		return
	}

	rows, err := app.DB.Query(query, searchRequest.Value)
	if err != nil {

		log.Printf("Ошибка поиска: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Ошибка выполнения поиска",
		})
		return
	}
	defer rows.Close()

	var stars []Stars
	for rows.Next() {
		var star Stars
		err := rows.Scan(&star.ID, &star.Name, &star.AlternativeName, &star.Constellation, &star.Distance, &star.Mass)
		if err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}
		stars = append(stars, star)
	}

	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: fmt.Sprintf("Найдено звезд: %d", len(stars)),
		Data:    stars,
	})
}

func main() {
	// Получаем конфигурацию
	config := getConfig()

	// Создаем экземпляр приложения
	app := &App{}

	// Настраиваем подключение к БД с использованием конфигурации
	if err := app.setupDatabase(config); err != nil {
		log.Fatal(err)
	}
	defer app.DB.Close() // закрываем соединение при завершении программы

	// Создаем роутер
	r := mux.NewRouter()

	// Регистрируем эндпоинты
	r.HandleFunc("/health", healthCheck).Methods("GET")

	r.HandleFunc("/api/stars/search", app.getStarSearch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/stars", app.getAllStars).Methods("GET")

	// Настройка CORS (для фронтенда)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Запускаем сервер на порту из конфигурации
	log.Printf("Сервер запущен на порту %s", config.Port)
	log.Printf("API доступно по адресу: http://localhost%s", config.Port)

	if err := http.ListenAndServe(config.Port, r); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
