# gowaker
An alarm clock written in go. Plays spotify, reads out the weather and calendar events.

## Deployment
### Server
```bash
$ make deploy
$ ssh waker
```
```bash
j@waker$ sudo systemctl stop gowaker
j@waker$ make pi
j@waker$ sudo systemctl start gowaker
```
See [scripts/README.md](scripts/README.md) for instructions on installing the systemd service.

## Testing
### Clear the DB
```bash
rm waker.db
```

### Set non-repeating alarm
```bash
curl -X POST localhost:8080/alarms -d '{"time":{"hour":9,"minute":52},"repeat":false}'
```

### Set repeating alarm
```bash
curl -X POST localhost:8080/alarms -d '{"time":{"hour":9,"minute":52},"repeat":true,"days":["sunday","monday","tuesday","wednesday","thursday","friday","saturday"]}'
```

### Get alarms
```bash
curl -X GET localhost:8080/alarms
```

### Set default playlist
```bash
curl -X GET localhost:8080/spotify/playlists
curl -X PUT localhost:8080/spotify/default_playlist -d '{"id":"3gMssemWp3VtdwMoZYSPc4"}'
```
