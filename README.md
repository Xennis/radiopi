# radiopi

## Local development

* Go to https://developer.spotify.com/dashboard and create a new app.
* Run the application:

```shell
go run main.go --client-id=<your-client-id> --client-secret=<your-client-secret> --device-id=<your-device-id> --playlist-run=<your-playlist-id>
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

### Autoplay on startup

```shell
# Copy the service and configure it:
cp radiopi.service-template radiopi.service
# Copy the service to the Pi:
scp radiopi.service pi@radiopi.local:
# On the Pi:
sudo mv radiopi /usr/local/bin/
sudo mv radiopi.service /etc/systemd/system/
sudo chmod 644 /etc/systemd/system/radiopi.service
sudo systemctl enable radiopi.service
sudo reboot
```
