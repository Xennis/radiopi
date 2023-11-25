package main

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	tokenFile = "token.json"
)

var (
	// staticContent holds the static web server content.
	//go:embed index.html
	staticContent embed.FS

	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(os.Getenv("SPOTIFY_REDIRECT_URI")),
		spotifyauth.WithScopes(spotifyauth.ScopeUserModifyPlaybackState),
	)
	state = ""
)

func randomState() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 25)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	tokByte, err := json.Marshal(tok)
	if err != nil {
		http.Error(w, "Couldn't marshal token", http.StatusInternalServerError)
		log.Fatal(err)
	}
	err = os.WriteFile(tokenFile, tokByte, 0644)
	if err != nil {
		http.Error(w, "Couldn't write token", http.StatusInternalServerError)
		log.Fatal(err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func main() {
	ctx := context.Background()

	deviceID := spotify.ID(os.Getenv("SPOTIFY_DEVICE_ID"))
	if deviceID == "" {
		log.Fatal("SPOTIFY_DEVICE_ID not set")
	}
	playlistURI := spotify.URI(os.Getenv("SPOTIFY_PLAYLIST_URI"))
	if playlistURI == "" {
		log.Fatal("SPOTIFY_PLAYLIST_URI not set")
	}

	if _, err := os.Stat(tokenFile); err == nil {
		tokenFile, err := os.Open(tokenFile)
		if err != nil {
			log.Fatal(err)
		}
		defer tokenFile.Close()
		fileContent, err := io.ReadAll(tokenFile)
		if err != nil {
			log.Fatal(err)
		}

		var tok oauth2.Token
		err = json.Unmarshal(fileContent, &tok)
		if err != nil {
			log.Fatal(err)
		}

		client := spotify.New(auth.Client(ctx, &tok))

		err = client.PlayOpt(ctx, &spotify.PlayOptions{
			DeviceID:        &deviceID,
			PlaybackContext: &playlistURI,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	http.Handle("/", http.FileServer(http.FS(staticContent)))
	http.HandleFunc("/callback", handleLogin)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		state = randomState()
		http.Redirect(w, r, auth.AuthURL(state), http.StatusTemporaryRedirect)
	})

	log.Print("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
