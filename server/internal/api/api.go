package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/guilherme25alves/ama-rocketseat-server/internal/store/pgstore"
	"github.com/jackc/pgx/v5"
)

type apiHandler struct {
	q           *pgstore.Queries // O ideal aqui era uso de interface (se fosse necessário mudar o banco, por exemplo, ia mudar implementação, mas poderíamos manter a mesma interface)
	r           *chi.Mux
	upgrader    websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mu          *sync.Mutex
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q: q,
		upgrader: websocket.Upgrader{
			// precisamos tratar CORs também, é usado CheckOrigin
			CheckOrigin: func(r *http.Request) bool {
				// Em condições normais, ideal é validar de fato, não dar bypass
				return true
			},
		},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mu:          &sync.Mutex{},
	}

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID, // atribuir ID para a requisição
		middleware.Recoverer, // se der erro em algum endpoint, para não matar o servidor e ele continuar rodando
		middleware.Logger)    // para permitir log do que está acontecendo nas requisições

	// Lidando com CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/subscribe/{room_id}", a.handleSubscribe)

	// com a estrutura abaixo vai criar endpoints do tipo: /api/rooms/
	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", a.handleCreateRoom)
			r.Get("/", a.handleGetRooms)

			// /api/rooms/{room_id}/messages
			r.Route("/{room_id}/messages", func(r chi.Router) {
				r.Post("/", a.handleCreateMessage)
				r.Get("/", a.handleGetRoomMessages)

				// /api/rooms/{room_id}/messages/{message_id}
				r.Route("/{message_id}", func(r chi.Router) {
					r.Get("/", a.handleGetRoomMessage)
					r.Patch("/react", a.handleReactToMessage)
					r.Delete("/react", a.handleRemoveReactFromMessage)
					r.Patch("/answer", a.handleMarkMessageAsAnswered)
				})
			})
		})
	})

	a.r = r
	return a
}

func (h apiHandler) handleGetRooms(w http.ResponseWriter, r *http.Request)               {}
func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request)        {}
func (h apiHandler) handleGetRoomMessage(w http.ResponseWriter, r *http.Request)         {}
func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request)             {}
func (h apiHandler) handleCreateMessage(w http.ResponseWriter, r *http.Request)          {}
func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request)         {}
func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request)  {}
func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {}

func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	rawRoomID := chi.URLParam(r, "room_id")
	roomID, err := uuid.Parse(rawRoomID)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	_, err = h.q.GetRoom(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "room not found! please, verify room id", http.StatusBadRequest)
			return
		}

		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("failed to upgrade connection", "error", err)
		http.Error(w, "failed to upgrade to websocket connection", http.StatusBadRequest)
		return
	}

	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())

	h.mu.Lock()
	if _, ok := h.subscribers[rawRoomID]; !ok {
		h.subscribers[rawRoomID] = make(map[*websocket.Conn]context.CancelFunc)

	}
	h.subscribers[rawRoomID][conn] = cancel
	h.mu.Unlock()

	<-ctx.Done()

	h.mu.Lock()
	delete(h.subscribers[rawRoomID], conn)
	h.mu.Unlock()
}
