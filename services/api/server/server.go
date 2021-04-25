package server

import (
	"net/http"
	"sort"

	"github.com/gempir/spamchamp/pkg/store"

	"github.com/gempir/spamchamp/pkg/config"
	"github.com/gempir/spamchamp/pkg/helix"
	"github.com/rs/cors"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// Server api server
type Server struct {
	cfg            *config.Config
	broadcastQueue chan BroadcastMessage
	helixClient    *helix.Client
	store          *store.Store
}

type BroadcastMessage struct {
	Records        []Record `json:"records"`
	JoinedChannels int      `json:"joinedChannels"`
	ActiveChannels int      `json:"activeChannels"`
}

type WordcloudWord struct {
	Text  string  `json:"text"`
	Value float64 `json:"value"`
}

type Record struct {
	Title  string  `json:"title"`
	Scores []Score `json:"scores"`
}

type Score struct {
	User  User    `json:"user"`
	Score float64 `json:"score"`
}

type User struct {
	Id             string `json:"id"`
	DisplayName    string `json:"displayName"`
	ProfilePicture string `json:"profilePicture"`
}

func (s *Record) GetScoresSorted() []Score {
	scores := s.Scores
	sort.SliceStable(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})
	return scores
}

// NewServer create api Server
func NewServer(cfg *config.Config, helixClient *helix.Client, store *store.Store, broadcastQueue chan BroadcastMessage) Server {
	return Server{
		cfg:            cfg,
		broadcastQueue: broadcastQueue,
		helixClient:    helixClient,
		store:          store,
	}
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var upgrader = websocket.Upgrader{}

func (s *Server) Start() {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	go s.handleMessages()
	go s.subscribeChannelPoints()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws", s.handleConnections)
	mux.HandleFunc("/api/redemption", s.handleChannelPointsRedemption)
	mux.HandleFunc("/api/oauth", s.handleOauth)

	handler := cors.Default().Handler(mux)
	log.Info("[api] listening on port :8035")
	err := http.ListenAndServe(":8035", handler)
	if err != nil {
		log.Fatal("[api] listenAndServe: ", err)
	}
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	clients[ws] = true
	// Make sure we close the connection when the function returns
	defer ws.Close()

	for {
		var msg BroadcastMessage
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Infof("[api] error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		s.broadcastQueue <- msg
	}
}

func (s *Server) handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-s.broadcastQueue
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Errorf("[api] error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}