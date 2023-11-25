# radiopi

## Local development

* Run `cp env.template .env` to create a `.env` file
* Configure the newly created `.env` file. For this go to https://developer.spotify.com/dashboard and create a new app.
* Run the application:

```shell
env $(cat .env | grep -v "#" | xargs) go run main.go
```

Go to http://localhost:3000 and login with your Spotify account.

## Build & copy to Raspberry Pi

Build for the Raspberry Pi:

```shell
env GOOS=linux GOARCH=arm GOARM=5 go build -o build/radiopi-arm5
```

(Optional) If you want you can add a new line to your `/etc/hosts` file with the IP of your Pi. For example:
```text
192.168.178.78  radiopi.local
```

Otherwise, please replace `radiopi.local` with the IP of your Pi in the following instructions.

Copy the binary to the Raspberry Pi
```shell
scp build/radiopi-arm5 pi@radiopi.local:radiopi
```
