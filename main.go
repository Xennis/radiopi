package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
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

	clientID     string
	clientSecret string

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

func saveToken(tok *oauth2.Token) error {
	bytes, err := json.Marshal(*tok)
	if err != nil {
		return fmt.Errorf("marshaling token: %w", err)
	}
	err = os.WriteFile(tokenFile, bytes, 0644)
	if err != nil {
		return fmt.Errorf("writing token: %w", err)
	}
	return nil
}

func loadToken() (*oauth2.Token, error) {
	file, err := os.Open(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("opening token file: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading token file: %w", err)
	}

	var tok oauth2.Token
	if err = json.Unmarshal(content, &tok); err != nil {
		return nil, fmt.Errorf("unmarshaling token: %w", err)
	}
	return &tok, nil
}

func handleLogin(auth *spotifyauth.Authenticator, w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("state mismatch: %s != %s\n", st, state)
	}
	if err := saveToken(tok); err != nil {
		http.Error(w, "couldn't save token", http.StatusInternalServerError)
		log.Fatal(err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// The Spotify API sometimes returns no error, but also does not start playing. To work around this, we
// check the current the player state.
func checkIsPlaying(ctx context.Context, client *spotify.Client, deviceID spotify.ID) error {
	ps, err := client.PlayerState(ctx)
	if err != nil {
		return fmt.Errorf("getting player state: %w", err)
	}
	if ps == nil {
		// It's possible that the API returns a '204 No Content' response, in which case the Go SDK returns
		// no error and a nil player state.
		return fmt.Errorf("nothing playing yet")
	}
	if ps.Device.ID != deviceID || !ps.Device.Active || !ps.Playing {
		return fmt.Errorf("nothing playing on the device %q yet", deviceID)
	}
	return nil
}

func main() {
	ctx := context.Background()

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
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserModifyPlaybackState,
		),
	)

	if _, err := os.Stat(tokenFile); err == nil {
		tok, err := loadToken()
		if err != nil {
			log.Fatalf("error loading token: %q", err)
		}
		client := spotify.New(auth.Client(ctx, tok))
		newTok, err := client.Token()
		if err != nil {
			log.Fatalf("error getting token: %q", err)
		}
		// store the refreshed token
		if err := saveToken(newTok); err != nil {
			log.Fatalf("error saving token: %q", err)
		}

		retries := 3
		for {
			log.Println("playing...")
			if err := client.PlayOpt(ctx, &spotify.PlayOptions{
				DeviceID:        &deviceID,
				PlaybackContext: &playlistURI,
			}); err != nil {
				if retries == 0 {
					log.Fatalf("error playing %q, giving up", err)
				}
				retries--
				log.Printf("error playing %q, retrying...", err)
				time.Sleep(10 * time.Second)
				continue
			}

			if err := checkIsPlaying(ctx, client, deviceID); err != nil {
				if retries == 0 {
					log.Fatalf("error is playing %q, giving up", err)
				}
				retries--
				log.Printf("error is playing %q, retrying...", err)
				time.Sleep(10 * time.Second)
				continue
			}
			break
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
