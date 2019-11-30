# Running as a systemd service
## Installing
1) Copy the service file to the systemd service dir
`sudo cp ~/gowaker/scripts/gowaker.service /etc/systemd/system/gowaker.service`
2) Enable the service
`sudo systemctl enable /etc/systemd/system/gowaker.service`
3) Start the service
`sudo systemctl start gowaker.service`

## Secrets
Secrets are managed by a systemd override. See https://serverfault.com/a/413408.
To update, run `sudo systemctl edit gowaker`

## Updating
If the systemd service file is updated:
1) Copy the updated file
`sudo cp ~/gowaker/scripts/gowaker.service /etc/systemd/system/gowaker.service`
2) Reload the daemon
`sudo systemctl daemon-reload`
