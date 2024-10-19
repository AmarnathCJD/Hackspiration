package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/julienschmidt/sse"
)

var (
	sseServer    = sse.New()
	activeFields = map[string]string{
		"team_id": "",
		"accept":  "false",
	}
	sseAdmin = sse.New()
	sseTeams = sse.New()

	CORS_HEADERS = map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
	}
)

func main() {
	http.Handle("/events", sseServer)
	http.Handle("/admin", sseAdmin)
	http.Handle("/team-sse", sseTeams)
	http.HandleFunc("/teams", teamsHandler)
	http.Handle("/int/team", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fi, err := os.Open("select-team.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fi.Close()
		http.ServeContent(w, r, "index.html", time.Now(), fi)
	}))
	http.Handle("/v", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fi, err := os.Open("vote.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fi.Close()
		http.ServeContent(w, r, "index.html", time.Now(), fi)
	}))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fi, err := os.Open("index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fi.Close()
		http.ServeContent(w, r, "index.html", time.Now(), fi)
	}))

	http.HandleFunc("/set-active-team", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]string
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		t, err := GetTeamByName(data["team_id"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		activeFields["team_id"] = data["team_id"]
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
			"team_id": activeFields["team_id"],
		})

		sseTeams.SendString("", "message", `{"team_name": "`+activeFields["team_id"]+`", "team_idea": "`+t.Project+`"}`)
	})

	http.HandleFunc("/get-active-team", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t, err := GetTeamByName(activeFields["team_id"])

		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{
				"team_name": activeFields["team_id"],
				"team_idea": "",
				"error":     err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"team_name": activeFields["team_id"],
			"team_idea": t.Project,
		})
	})

	http.HandleFunc("/add-team", func(w http.ResponseWriter, r *http.Request) {
		var team Team
		var mapTeam map[string]string
		err := json.NewDecoder(r.Body).Decode(&mapTeam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		team.Name = mapTeam["name"]
		team.Project = mapTeam["topic"]

		err = team.InsertTeam()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	})

	http.HandleFunc("/remove-team", func(w http.ResponseWriter, r *http.Request) {
		var mapTeam map[string]string
		err := json.NewDecoder(r.Body).Decode(&mapTeam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		DeleteTeam(mapTeam["team_id"])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	})

	http.HandleFunc("/set-accept", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]bool
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		activeFields["accept"] = strconv.FormatBool(data["accept"])
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
			"accept":  activeFields["accept"],
		})
	})

	http.HandleFunc("/get-accept", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, err := strconv.ParseBool(activeFields["accept"])
		if err != nil {
			b = false
		}
		json.NewEncoder(w).Encode(map[string]string{
			"accept": strconv.FormatBool(b),
		})
	})

	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {
		if activeFields["accept"] != "true" {
			http.Error(w, "Voting is not currently open", http.StatusBadRequest)
			return
		}
		ip := getIp(r)
		var data map[string]any
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		teamName := data["teamName"].(string)
		feedback := data["feedback"].(string)
		rating := data["rating"].(float64)
		sender := data["sender"].(string)
		if HasIpVoted(ip, teamName) {
			http.Error(w, "You have already voted for this team", http.StatusBadRequest)
			sseAdmin.SendString("", "message", "Attempted vote from "+ip+" for "+teamName+" but already voted")
			return
		}

		vote := Vote{
			IpAddr:   ip,
			TeamName: teamName,
			Feedback: feedback,
			Rating:   rating,
			Sender:   sender,
		}
		err = vote.InsertVote()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			sseAdmin.SendString("", "message", "('"+ip+": "+err.Error()+"')")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})

		sseAdmin.SendString("", "message", "New vote for <b>"+teamName+"</b> from "+ip+"<br><br><code>{'feedback': "+feedback+", 'rating': "+strconv.FormatFloat(rating, 'f', -1, 64)+", 'sender': "+sender+"}</code>")
	})

	go LiveEventSwitcher()

	http.ListenAndServe(":8080", nil)
}

func getIp(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func LiveEventSwitcher() {
	for {
		//sseServer.SendString("", "message", "Hello, world!")
		time.Sleep(1 * time.Second)
	}
}

func teamsHandler(w http.ResponseWriter, r *http.Request) {
	teams, err := GetTeams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}
