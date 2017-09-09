## go-messenger-bot
Facebook Messenger Quick-Start Sample in Golang https://glitch.com/edit/#!/project/messenger-bot

### building and running

`go build messenger-bot.go`

`VERIFY_TOKEN=??? PAGE_ACCESS_TOKEN=??? ./messenger-bot`

### using docker
`docker run --publish 8080:8080 --env VERIFY_TOKEN=??? --env PAGE_ACCESS_TOKEN=??? siriusdely/go-messenger-bot`
