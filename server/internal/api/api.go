package api

import (
	"context"
	"encoding/json"
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

func (h apiHandler) handleGetRooms(w http.ResponseWriter, r *http.Request)        {}
func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request) {}
func (h apiHandler) handleGetRoomMessage(w http.ResponseWriter, r *http.Request)  {}

func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	type _body struct {
		Theme string `json:"theme"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	roomID, err := h.q.InsertRoom(r.Context(), body.Theme)
	if err != nil {
		slog.Error("failed to insert room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	data, _ := json.Marshal(response{ID: roomID.String()})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)

}

func (h apiHandler) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	rawRoomID, roomID, shouldReturn := handleValidateRoomID(r, w, h)
	if shouldReturn {
		return
	}

	type _body struct {
		Message string `json:"message"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	messageID, err := h.q.InsertMessage(r.Context(), pgstore.InsertMessageParams{
		RoomID:  roomID,
		Message: body.Message,
	})

	if err != nil {
		slog.Error("failed to insert message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	data, _ := json.Marshal(response{ID: messageID.String()})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)

	go h.notifyClients(Message{
		Kind:   MessageKindMessageCreated,
		RoomID: rawRoomID,
		Value: MessageMessageCreated{
			ID:      messageID.String(),
			Message: body.Message,
		},
	})
}

func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request)         {}
func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request)  {}
func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {}

const (
	MessageKindMessageCreated = "message_created"
)

type MessageMessageCreated struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind   string `json:"kind"`
	Value  any    `json:"value"`
	RoomID string `json:"-"` // quando colocamos um "-", indicamos ao pacote  json do GO que não queremos encoder do campo
}

func (h apiHandler) notifyClients(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribers, ok := h.subscribers[msg.RoomID]

	// não tem ninguém para notificar, apenas retorna
	if !ok || len(subscribers) == 0 {
		return
	}

	// notifica todos os clientes na sala
	for conn, cancel := range subscribers {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Error("faile to send message to client", "error", err)
			cancel()
		}
	}

}

func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	rawRoomID, _, shouldReturn := handleValidateRoomID(r, w, h)
	if shouldReturn {
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
	slog.Info("new client connected", "room_id", rawRoomID, "client_ip", r.RemoteAddr)
	h.subscribers[rawRoomID][conn] = cancel
	h.mu.Unlock()

	<-ctx.Done()

	h.mu.Lock()
	delete(h.subscribers[rawRoomID], conn)
	h.mu.Unlock()
}

func handleValidateRoomID(r *http.Request, w http.ResponseWriter, h apiHandler) (string, uuid.UUID, bool) {
	rawRoomID := chi.URLParam(r, "room_id")
	roomID, err := uuid.Parse(rawRoomID)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return "", uuid.Nil, true
	}

	_, err = h.q.GetRoom(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "room not found! please, verify room id", http.StatusBadRequest)
			return "", uuid.Nil, true
		}

		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return "", uuid.Nil, true
	}
	return rawRoomID, roomID, false
}
