# Testing
## Clear the DB
```bash
rm waker.db
```

## Set non-repeating alarm
```bash
curl -X POST localhost:8080/alarms -d '{"time":{"hour":9,"minute":52},"repeat":false}'
```

## Set repeating alarm
```bash
curl -X POST localhost:8080/alarms -d '{"time":{"hour":9,"minute":52},"repeat":true,"days":["sunday","monday","tuesday","wednesday","thursday","friday","saturday"]}'
```

## Set default playlist
```bash
curl -X GET localhost:8080/spotify/playlists
curl -X PUT localhost:8080/spotify/default_playlist -d '{"time":{"hour":9,"minute":52},"repeat":true,"days":["sunday","monday","tuesday","wednesday","thursday","friday","saturday"]}'
```
