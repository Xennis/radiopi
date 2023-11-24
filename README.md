# radiopi

## Local development

* Run `cp env.template .env` to create a `.env` file
* Configure the newly created `.env` file. For this go to https://developer.spotify.com/dashboard and create a new app.
* Run the application:

```shell
env $(cat .env | grep -v "#" | xargs) go run main.go
```

Go to http://localhost:3000 and login with your Spotify account.

## Build

Build for the Raspberry Pi:

```shell
env GOOS=linux GOARCH=arm GOARM=5 go build -o build/radiopi-arm5
```
