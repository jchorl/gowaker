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
