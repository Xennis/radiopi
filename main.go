package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const (
	redirectURI = "http://localhost:3000/callback"
	tokenFile   = "token.json"
)

var (
	// staticContent holds the static web server content.
	//go:embed index.html
	staticContent embed.FS

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

func handleLogin(auth *spotifyauth.Authenticator, w http.ResponseWriter, r *http.Request) {
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

	clientIDPtr := flag.String("client-id", "", "Spotify client ID")
	deviceSecretPtr := flag.String("client-secret", "", "Spotify client secret")
	deviceIDPtr := flag.String("device-id", "", "Spotify device ID")
	playlistURIPtr := flag.String("playlist-uri", "", "Spotify playlist URI in the form spotify:playlist:<id>")
	flag.Parse()

	deviceID := spotify.ID(*deviceIDPtr)
	if deviceID == "" {
		log.Fatal("device-id not set")
	}
	playlistURI := spotify.URI(*playlistURIPtr)
	if playlistURI == "" {
		log.Fatal("playlist-uri not set")
	}
	auth := spotifyauth.New(
		spotifyauth.WithClientID(*clientIDPtr),
		spotifyauth.WithClientSecret(*deviceSecretPtr),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserModifyPlaybackState),
	)

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

		retries := 3
		for {
			err = client.PlayOpt(ctx, &spotify.PlayOptions{
				DeviceID:        &deviceID,
				PlaybackContext: &playlistURI,
			})
			if err == nil {
				break
			}
			// Raspotify (Spotify Connect) takes a while to start up, so retry a few times.
			if (retries > 0) && err.Error() == "Device not found" {
				retries--
				log.Printf("error playing %q, retrying in 10 seconds...", err)
				time.Sleep(10 * time.Second)
			} else {
				log.Fatalf("error playing %q, giving up", err)
			}
		}
	}

	http.Handle("/", http.FileServer(http.FS(staticContent)))
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(auth, w, r)
	})
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
