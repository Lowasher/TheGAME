package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Message struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

type Player struct {
	Name     string    `json:"name"`
	HP       int       `json:"hp"`
	MaxHP    int       `json:"max_hp"`
	Level    int       `json:"level"`
	Strength int       `json:"strength"`
	TargetID string    `json:"target_id"` // Имя противника
	LastSeen time.Time `json:"-"`
}

type BattleMove struct {
	Attack string `json:"attack"` // head, torso, legs
	Defend string `json:"defend"` // head, torso, legs
}

type GameState struct {
	Players  map[string]*Player `json:"players"`
	Messages []Message          `json:"messages"`
}

var (
	state = GameState{
		Players:  make(map[string]*Player),
		Messages: []Message{{Author: "Система", Text: "Мир запущен!", Time: ""}},
	}
	// Хранилище ходов: [PlayerName]BattleMove
	pendingMoves = make(map[string]BattleMove)
	mu           sync.Mutex
)

func main() {
	http.HandleFunc("/sync", handleSync)
	http.HandleFunc("/chat", handleChat)
	http.HandleFunc("/move", handleMove)

	fmt.Println("Сервер PvP запущен на :8080")
	http.ListenAndServe(":8080", nil)
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	if r.Method == http.MethodPost {
		var p Player
		if err := json.NewDecoder(r.Body).Decode(&p); err == nil {
			p.LastSeen = time.Now()
			state.Players[p.Name] = &p
		}
	}
	json.NewEncoder(w).Encode(state)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err == nil {
			mu.Lock()
			msg.Time = time.Now().Format("15:04")
			state.Messages = append(state.Messages, msg)
			mu.Unlock()
		}
	}
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var req struct {
		PlayerName string     `json:"player_name"`
		Move       BattleMove `json:"move"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}

	pendingMoves[req.PlayerName] = req.Move
	p1 := state.Players[req.PlayerName]
	p2Name := p1.TargetID
	
	// Если противник тоже походил — считаем результат
	if m2, ok := pendingMoves[p2Name]; ok {
		m1 := req.Move
		p2 := state.Players[p2Name]

		// Логика: если атака p1 не совпала с защитой p2 -> урон
		res1 := calculateDamage(p1, p2, m1.Attack, m2.Defend)
		res2 := calculateDamage(p2, p1, m2.Attack, m1.Defend)

		state.Messages = append(state.Messages, Message{
			Author: "Бой",
			Text:   fmt.Sprintf("%s -> %s (%d), %s -> %s (%d)", p1.Name, p2.Name, res1, p2.Name, p1.Name, res2),
			Time:   time.Now().Format("15:04"),
		})

		delete(pendingMoves, p1.Name)
		delete(pendingMoves, p2Name)
	}
}

func calculateDamage(att *Player, def *Player, atkPart, defPart string) int {
	if atkPart == defPart {
		return 0 // Заблокировано
	}
	dmg := att.Strength
	def.HP -= dmg
	if def.HP < 0 { def.HP = 0 }
	return dmg
}